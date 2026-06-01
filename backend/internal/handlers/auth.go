package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ledger-backend/internal/config"
	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
	"ledger-backend/internal/utils"
)

func (h *AppHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	if locked, until := h.loginLocked(req.Username, c.ClientIP()); locked {
		msg := fmt.Sprintf("登录失败次数过多，请于 %s 后再试", until.Format("2006-01-02 15:04:05"))
		h.audit(c, nil, nil, "login_blocked", "auth_throttle", "", false, models.JSONMap{"username": req.Username, "ip": c.ClientIP(), "locked_until": until.Format(time.RFC3339)})
		fail(c, http.StatusTooManyRequests, msg)
		return
	}

	var user models.User
	err := h.db.Preload("Family").Where("username = ?", req.Username).First(&user).Error
	passwordOK := err == nil && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) == nil
	if err != nil || !passwordOK {
		h.recordLoginFailure(req.Username, c.ClientIP())
		h.audit(c, nil, nil, "login_failed", "user", "", false, models.JSONMap{"username": req.Username})
		fail(c, http.StatusUnauthorized, "账号或密码错误")
		return
	}
	if user.Status != models.StatusEnabled {
		h.audit(c, user.FamilyID, &user.ID, "login_failed", "user", strconv.Itoa(int(user.ID)), false, models.JSONMap{"reason": "disabled"})
		fail(c, http.StatusForbidden, "账号已禁用")
		return
	}
	if user.Role != models.RoleSuperAdmin && (user.Family == nil || user.Family.Status != models.StatusEnabled) {
		fail(c, http.StatusForbidden, "所属家庭已禁用")
		return
	}
	h.clearLoginFailures(req.Username, c.ClientIP())

	accessToken, err := utils.GenerateAccessToken(user.ID, user.FamilyID, user.Username, user.Role)
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成访问令牌失败")
		return
	}
	refreshToken, refreshHash, err := utils.NewRefreshToken()
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成刷新令牌失败")
		return
	}
	now := time.Now()
	session := models.UserSession{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        now.AddDate(0, 0, int(config.AppConfig.JWT.RefreshTokenDays)),
		IP:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}
	if err := h.db.Create(&session).Error; err != nil {
		fail(c, http.StatusInternalServerError, "保存会话失败")
		return
	}
	h.db.Model(&user).Update("last_login_at", now)
	h.audit(c, user.FamilyID, &user.ID, "login_success", "user", strconv.Itoa(int(user.ID)), true, nil)

	ok(c, gin.H{
		"access_token":         accessToken,
		"refresh_token":        refreshToken,
		"token_type":           "Bearer",
		"expires_in":           int64(config.AppConfig.JWT.AccessTokenHours * 3600),
		"must_change_password": user.MustChangePassword,
		"user":                 sanitizeUser(user),
	})
}

func (h *AppHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	var session models.UserSession
	if err := h.db.Where("refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?", utils.HashToken(req.RefreshToken), time.Now()).First(&session).Error; err != nil {
		fail(c, http.StatusUnauthorized, "刷新令牌无效或已过期")
		return
	}
	var user models.User
	if err := h.db.First(&user, session.UserID).Error; err != nil || user.Status != models.StatusEnabled {
		fail(c, http.StatusUnauthorized, "账号不可用")
		return
	}
	accessToken, err := utils.GenerateAccessToken(user.ID, user.FamilyID, user.Username, user.Role)
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成访问令牌失败")
		return
	}
	ok(c, gin.H{"access_token": accessToken, "token_type": "Bearer", "expires_in": int64(config.AppConfig.JWT.AccessTokenHours * 3600)})
}

func (h *AppHandler) Logout(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	now := time.Now()
	h.db.Model(&models.UserSession{}).Where("refresh_token_hash = ?", utils.HashToken(req.RefreshToken)).Update("revoked_at", now)
	h.audit(c, auth.FamilyID, &auth.UserID, "logout", "user_session", "", true, nil)
	ok(c, nil)
}

func (h *AppHandler) Me(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var user models.User
	if err := h.db.Preload("Family").First(&user, auth.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	ok(c, sanitizeUser(user))
}

func (h *AppHandler) ChangePassword(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	var user models.User
	if err := h.db.First(&user, auth.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, "用户不存在")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		fail(c, http.StatusBadRequest, "原密码错误")
		return
	}
	if msg := validatePassword(req.NewPassword); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	now := time.Now()
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(map[string]any{"password_hash": string(hash), "must_change_password": false}).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", now).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "修改密码失败")
		return
	}
	h.audit(c, user.FamilyID, &user.ID, "change_password", "user", strconv.Itoa(int(user.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) loginLocked(username, ip string) (bool, time.Time) {
	now := time.Now()
	keys := loginThrottleKeys(username, ip)
	if len(keys) == 0 {
		return false, time.Time{}
	}
	var locks []models.AuthThrottle
	condition, args := loginThrottleCondition(keys)
	h.db.Where(condition, args...).Where("locked_until IS NOT NULL AND locked_until > ?", now).Find(&locks)
	var latest time.Time
	for _, lock := range locks {
		if lock.LockedUntil != nil && lock.LockedUntil.After(latest) {
			latest = *lock.LockedUntil
		}
	}
	return !latest.IsZero(), latest
}

func (h *AppHandler) recordLoginFailure(username, ip string) {
	now := time.Now()
	cfg := config.AppConfig.Security
	window := time.Duration(cfg.LoginWindowMinutes) * time.Minute
	lockDuration := time.Duration(cfg.LoginLockMinutes) * time.Minute
	for _, item := range loginThrottleKeys(username, ip) {
		var throttle models.AuthThrottle
		err := h.db.Where("scope = ? AND key = ?", item[0], item[1]).First(&throttle).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			throttle = models.AuthThrottle{Scope: item[0], Key: item[1], FailedCount: 1, WindowStartedAt: now, LastFailedAt: now}
			if throttle.FailedCount >= cfg.LoginMaxAttempts {
				lockedUntil := now.Add(lockDuration)
				throttle.LockedUntil = &lockedUntil
			}
			h.db.Create(&throttle)
			continue
		}
		if err != nil {
			continue
		}
		if now.Sub(throttle.WindowStartedAt) > window {
			throttle.FailedCount = 0
			throttle.WindowStartedAt = now
			throttle.LockedUntil = nil
		}
		throttle.FailedCount += 1
		throttle.LastFailedAt = now
		if throttle.FailedCount >= cfg.LoginMaxAttempts {
			lockedUntil := now.Add(lockDuration)
			throttle.LockedUntil = &lockedUntil
		}
		h.db.Save(&throttle)
	}
}

func (h *AppHandler) clearLoginFailures(username, ip string) {
	keys := loginThrottleKeys(username, ip)
	if len(keys) == 0 {
		return
	}
	condition, args := loginThrottleCondition(keys)
	h.db.Where(condition, args...).Delete(&models.AuthThrottle{})
}

func loginThrottleKeys(username, ip string) [][2]string {
	keys := [][2]string{}
	if key := normalizeLoginKey(username); key != "" {
		keys = append(keys, [2]string{"username", key})
	}
	if key := strings.TrimSpace(ip); key != "" {
		keys = append(keys, [2]string{"ip", key})
	}
	return keys
}

func normalizeLoginKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func loginThrottleCondition(keys [][2]string) (string, []any) {
	parts := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)*2)
	for _, item := range keys {
		parts = append(parts, "(scope = ? AND key = ?)")
		args = append(args, item[0], item[1])
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}
