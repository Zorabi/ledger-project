package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ledger-backend/internal/config"
	"ledger-backend/internal/exporter"
	"ledger-backend/internal/importer"
	"ledger-backend/internal/mailer"
	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
	"ledger-backend/internal/utils"
)

type AppHandler struct {
	db *gorm.DB
}

func NewAppHandler(db *gorm.DB) *AppHandler {
	return &AppHandler{db: db}
}

func (h *AppHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) { ok(c, gin.H{"status": "ok"}) })
	v1.GET("/version", h.Version)

	auth := v1.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)

	protected := v1.Group("").Use(middleware.AuthRequired())
	protected.GET("/me", h.Me)
	protected.POST("/auth/logout", h.Logout)
	protected.POST("/auth/change-password", h.ChangePassword)

	protected.GET("/families", h.ListFamilies)
	protected.POST("/families", middleware.RequireRoles(models.RoleSuperAdmin), h.CreateFamily)
	protected.PUT("/families/:id", middleware.RequireRoles(models.RoleSuperAdmin), h.UpdateFamily)
	protected.PATCH("/families/:id/status", middleware.RequireRoles(models.RoleSuperAdmin), h.UpdateFamilyStatus)

	protected.GET("/users", h.ListUsers)
	protected.POST("/users", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.CreateUser)
	protected.PUT("/users/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.UpdateUser)
	protected.PATCH("/users/:id/status", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.UpdateUserStatus)
	protected.POST("/users/:id/reset-password", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ResetPassword)

	protected.GET("/dictionary-types", h.ListDictionaryTypes)
	protected.POST("/dictionary-types", middleware.RequireRoles(models.RoleSuperAdmin), h.CreateDictionaryType)
	protected.PUT("/dictionary-types/:id", middleware.RequireRoles(models.RoleSuperAdmin), h.UpdateDictionaryType)
	protected.GET("/dictionary-items", h.ListDictionaryItems)
	protected.POST("/dictionary-items", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.CreateDictionaryItem)
	protected.PUT("/dictionary-items/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.UpdateDictionaryItem)
	protected.DELETE("/dictionary-items/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.DeleteDictionaryItem)

	protected.GET("/contacts", h.ListContacts)
	protected.POST("/contacts", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.CreateContact)
	protected.GET("/contacts/:id", h.GetContact)
	protected.PUT("/contacts/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.UpdateContact)
	protected.DELETE("/contacts/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.DeleteContact)

	protected.GET("/gift-records", h.ListGiftRecords)
	protected.POST("/gift-records", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.CreateGiftRecord)
	protected.GET("/gift-records/:id", h.GetGiftRecord)
	protected.PUT("/gift-records/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.UpdateGiftRecord)
	protected.DELETE("/gift-records/:id", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin, models.RoleMember), h.DeleteGiftRecord)

	protected.GET("/stats/overview", h.StatsOverview)
	protected.GET("/audit-logs", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ListAuditLogs)
	protected.GET("/backup/config", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.GetBackupConfig)
	protected.PUT("/backup/config", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.UpdateBackupConfig)
	protected.POST("/backup/export", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ManualBackup)
	protected.GET("/backup/jobs", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ListBackupJobs)
	protected.GET("/imports/template", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.DownloadImportTemplate)
	protected.POST("/imports/dry-run", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ImportDryRun)
	protected.GET("/imports", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ListImportJobs)
	protected.GET("/imports/:id/errors", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ListImportErrors)
	protected.POST("/imports/:id/execute", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ExecuteImport)
}

func (h *AppHandler) Version(c *gin.Context) {
	ok(c, gin.H{
		"name":    config.AppConfig.App.Name,
		"version": config.AppConfig.App.Version,
	})
}

func (h *AppHandler) SeedDefaults() error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(config.AppConfig.Bootstrap.SuperAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return h.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("role = ?", models.RoleSuperAdmin).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			admin := models.User{
				Username:           config.AppConfig.Bootstrap.SuperAdminUsername,
				DisplayName:        "系统管理员",
				PasswordHash:       string(passwordHash),
				Role:               models.RoleSuperAdmin,
				Status:             models.StatusEnabled,
				MustChangePassword: true,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
		}
		return seedDictionary(tx)
	})
}

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

func (h *AppHandler) ListFamilies(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var families []models.Family
	q := h.db.Model(&models.Family{})
	if auth.Role != models.RoleSuperAdmin {
		if auth.FamilyID == nil {
			fail(c, http.StatusForbidden, "当前账号未归属家庭")
			return
		}
		q = q.Where("id = ?", *auth.FamilyID)
	}
	pageNum, pageSize := paginationParams(c)
	var total int64
	q.Count(&total)
	q.Order("id desc").Offset((pageNum - 1) * pageSize).Limit(pageSize).Find(&families)
	fillFamilyAdmins(h.db, families)
	ok(c, gin.H{"total": total, "list": families, "page": pageNum, "page_size": pageSize})
}

func (h *AppHandler) CreateFamily(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		ContactName  string `json:"contact_name"`
		ContactEmail string `json:"contact_email"`
		Remark       string `json:"remark"`
		AdminUserID  *uint  `json:"admin_user_id"`
		Admin        *struct {
			Username    string `json:"username" binding:"required"`
			DisplayName string `json:"display_name" binding:"required"`
			Email       string `json:"email"`
			Password    string `json:"password" binding:"required"`
		} `json:"admin"`
	}
	if !bind(c, &req) {
		return
	}
	auth, _ := middleware.CurrentUser(c)
	family := models.Family{Name: req.Name, ContactName: req.ContactName, ContactEmail: req.ContactEmail, Remark: req.Remark, Status: models.StatusEnabled}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&family).Error; err != nil {
			return err
		}
		if req.AdminUserID != nil {
			var admin models.User
			if err := tx.Where("id = ? AND role <> ?", *req.AdminUserID, models.RoleSuperAdmin).First(&admin).Error; err != nil {
				return fmt.Errorf("选择的管理员用户不存在")
			}
			if admin.FamilyID != nil && *admin.FamilyID != family.ID {
				return fmt.Errorf("选择的管理员已归属其他家庭")
			}
			if err := tx.Model(&admin).Updates(map[string]any{"family_id": family.ID, "role": models.RoleFamilyAdmin}).Error; err != nil {
				return err
			}
			if err := tx.Model(&family).Update("admin_user_id", admin.ID).Error; err != nil {
				return err
			}
			family.AdminUserID = &admin.ID
		} else if req.Admin != nil {
			if msg := validatePassword(req.Admin.Password); msg != "" {
				return errors.New(msg)
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
			admin := models.User{
				FamilyID:           &family.ID,
				Username:           req.Admin.Username,
				DisplayName:        req.Admin.DisplayName,
				Email:              req.Admin.Email,
				PasswordHash:       string(hash),
				Role:               models.RoleFamilyAdmin,
				Status:             models.StatusEnabled,
				MustChangePassword: true,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
			if err := tx.Model(&family).Update("admin_user_id", admin.ID).Error; err != nil {
				return err
			}
			family.AdminUserID = &admin.ID
		}
		return nil
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &family.ID, &auth.UserID, "create_family", "family", strconv.Itoa(int(family.ID)), true, nil)
	ok(c, family)
}

func (h *AppHandler) UpdateFamily(c *gin.Context) {
	var family models.Family
	if !findByID(c, h.db, &family) {
		return
	}
	var req struct {
		Name         string `json:"name" binding:"required"`
		ContactName  string `json:"contact_name"`
		ContactEmail string `json:"contact_email"`
		Remark       string `json:"remark"`
		AdminUserID  *uint  `json:"admin_user_id"`
		Admin        *struct {
			Username    string `json:"username" binding:"required"`
			DisplayName string `json:"display_name" binding:"required"`
			Email       string `json:"email"`
			Password    string `json:"password" binding:"required"`
		} `json:"admin"`
	}
	if !bind(c, &req) {
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"name": req.Name, "contact_name": req.ContactName, "contact_email": req.ContactEmail, "remark": req.Remark}
		if err := tx.Model(&family).Updates(updates).Error; err != nil {
			return err
		}
		if req.AdminUserID != nil {
			var admin models.User
			if err := tx.Where("id = ? AND role <> ?", *req.AdminUserID, models.RoleSuperAdmin).First(&admin).Error; err != nil {
				return fmt.Errorf("选择的管理员用户不存在")
			}
			if admin.FamilyID != nil && *admin.FamilyID != family.ID {
				return fmt.Errorf("选择的管理员已归属其他家庭")
			}
			if err := tx.Model(&admin).Updates(map[string]any{"family_id": family.ID, "role": models.RoleFamilyAdmin}).Error; err != nil {
				return err
			}
			return tx.Model(&family).Update("admin_user_id", admin.ID).Error
		}
		if req.Admin != nil {
			if msg := validatePassword(req.Admin.Password); msg != "" {
				return errors.New(msg)
			}
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
			admin := models.User{FamilyID: &family.ID, Username: req.Admin.Username, DisplayName: req.Admin.DisplayName, Email: req.Admin.Email, PasswordHash: string(hash), Role: models.RoleFamilyAdmin, Status: models.StatusEnabled, MustChangePassword: true}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
			return tx.Model(&family).Update("admin_user_id", admin.ID).Error
		}
		return nil
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	out := []models.Family{family}
	fillFamilyAdmins(h.db, out)
	ok(c, out[0])
}

func (h *AppHandler) UpdateFamilyStatus(c *gin.Context) {
	var family models.Family
	if !findByID(c, h.db, &family) {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Status != models.StatusEnabled && req.Status != models.StatusDisabled {
		fail(c, http.StatusBadRequest, "状态只能是 enabled 或 disabled")
		return
	}
	h.db.Model(&family).Update("status", req.Status)
	ok(c, nil)
}

func (h *AppHandler) ListUsers(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var users []models.User
	q := h.db.Preload("Family").Model(&models.User{})
	if auth.Role != models.RoleSuperAdmin {
		if auth.FamilyID == nil {
			fail(c, http.StatusForbidden, "当前账号未归属家庭")
			return
		}
		q = q.Where("family_id = ?", *auth.FamilyID)
	}
	page(c, q.Order("id desc"), &users)
}

func (h *AppHandler) CreateUser(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var req struct {
		FamilyID    *uint  `json:"family_id"`
		Username    string `json:"username" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
		Email       string `json:"email"`
		Password    string `json:"password" binding:"required"`
		Role        string `json:"role" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	role := models.NormalizeRole(req.Role)
	if auth.Role == models.RoleFamilyAdmin {
		if role == models.RoleSuperAdmin || req.FamilyID != nil && *req.FamilyID != *auth.FamilyID {
			fail(c, http.StatusForbidden, "家庭管理员只能创建本家庭成员或管理员")
			return
		}
		req.FamilyID = auth.FamilyID
	}
	if role != models.RoleFamilyAdmin && role != models.RoleMember && role != models.RoleSuperAdmin {
		fail(c, http.StatusBadRequest, "角色不合法")
		return
	}
	if role != models.RoleSuperAdmin && req.FamilyID == nil {
		fail(c, http.StatusBadRequest, "非系统管理员必须指定家庭")
		return
	}
	if msg := validatePassword(req.Password); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{FamilyID: req.FamilyID, Username: req.Username, DisplayName: req.DisplayName, Email: req.Email, PasswordHash: string(hash), Role: role, Status: models.StatusEnabled, MustChangePassword: true}
	if err := h.db.Create(&user).Error; err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, user.FamilyID, &auth.UserID, "create_user", "user", strconv.Itoa(int(user.ID)), true, nil)
	ok(c, sanitizeUser(user))
}

func (h *AppHandler) UpdateUser(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var user models.User
	if !scopedUser(c, h.db, auth, &user) {
		return
	}
	var req struct {
		Username    string `json:"username" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
		Email       string `json:"email"`
		Role        string `json:"role" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	role := models.NormalizeRole(req.Role)
	if auth.Role == models.RoleFamilyAdmin && role == models.RoleSuperAdmin {
		fail(c, http.StatusForbidden, "家庭管理员不能设置系统管理员")
		return
	}
	if role != models.RoleFamilyAdmin && role != models.RoleMember && role != models.RoleSuperAdmin {
		fail(c, http.StatusBadRequest, "角色不合法")
		return
	}
	if err := h.db.Model(&user).Updates(map[string]any{"username": req.Username, "display_name": req.DisplayName, "email": req.Email, "role": role}).Error; err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, user.FamilyID, &auth.UserID, "update_user", "user", strconv.Itoa(int(user.ID)), true, nil)
	ok(c, sanitizeUser(user))
}

func (h *AppHandler) UpdateUserStatus(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var user models.User
	if !scopedUser(c, h.db, auth, &user) {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Status != models.StatusEnabled && req.Status != models.StatusDisabled {
		fail(c, http.StatusBadRequest, "状态只能是 enabled 或 disabled")
		return
	}
	now := time.Now()
	h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("status", req.Status).Error; err != nil {
			return err
		}
		if req.Status == models.StatusDisabled {
			return tx.Model(&models.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", now).Error
		}
		return nil
	})
	h.audit(c, user.FamilyID, &auth.UserID, "update_user_status", "user", strconv.Itoa(int(user.ID)), true, models.JSONMap{"status": req.Status})
	ok(c, nil)
}

func (h *AppHandler) ResetPassword(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var user models.User
	if !scopedUser(c, h.db, auth, &user) {
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if !bind(c, &req) {
		return
	}
	if msg := validatePassword(req.Password); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	now := time.Now()
	h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(map[string]any{"password_hash": string(hash), "must_change_password": true}).Error; err != nil {
			return err
		}
		return tx.Model(&models.UserSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", now).Error
	})
	h.clearLoginFailures(user.Username, "")
	h.audit(c, user.FamilyID, &auth.UserID, "reset_password", "user", strconv.Itoa(int(user.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) ListDictionaryTypes(c *gin.Context) {
	var items []models.DictionaryType
	page(c, h.db.Model(&models.DictionaryType{}).Order("sort_order asc,id asc"), &items)
}

func (h *AppHandler) CreateDictionaryType(c *gin.Context) {
	var req models.DictionaryType
	if !bind(c, &req) {
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, req)
}

func (h *AppHandler) UpdateDictionaryType(c *gin.Context) {
	var item models.DictionaryType
	if !findByID(c, h.db, &item) {
		return
	}
	var req models.DictionaryType
	if !bind(c, &req) {
		return
	}
	h.db.Model(&item).Updates(map[string]any{"code": req.Code, "name": req.Name, "description": req.Description, "allow_family_custom": req.AllowFamilyCustom, "sort_order": req.SortOrder, "enabled": req.Enabled})
	auth, _ := middleware.CurrentUser(c)
	h.audit(c, nil, &auth.UserID, "update_dictionary_type", "dictionary_type", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, item)
}

func (h *AppHandler) ListDictionaryItems(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	typeCode := c.Query("type")
	var dtype models.DictionaryType
	if typeCode != "" {
		if err := h.db.Where("code = ?", typeCode).First(&dtype).Error; err != nil {
			fail(c, http.StatusNotFound, "字典类型不存在")
			return
		}
	}
	q := h.db.Preload("Type").Model(&models.DictionaryItem{}).Where("enabled = true")
	if dtype.ID != 0 {
		q = q.Where("type_id = ?", dtype.ID)
	}
	if auth.Role == models.RoleSuperAdmin {
		if familyID := c.Query("family_id"); familyID != "" {
			q = q.Where("family_id IS NULL OR family_id = ?", familyID)
		} else {
			q = q.Where("family_id IS NULL")
		}
	} else if auth.FamilyID != nil {
		q = q.Where("family_id IS NULL OR family_id = ?", *auth.FamilyID)
	}
	var items []models.DictionaryItem
	page(c, q.Order("sort_order asc,id asc"), &items)
}

func (h *AppHandler) CreateDictionaryItem(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var req models.DictionaryItem
	if !bind(c, &req) {
		return
	}
	if auth.Role == models.RoleSuperAdmin {
		req.IsSystem = req.FamilyID == nil
	} else {
		req.FamilyID = auth.FamilyID
		req.IsSystem = false
	}
	if err := h.db.Create(&req).Error; err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, req.FamilyID, &auth.UserID, "create_dictionary_item", "dictionary_item", strconv.Itoa(int(req.ID)), true, nil)
	ok(c, req)
}

func (h *AppHandler) UpdateDictionaryItem(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.DictionaryItem
	if !scopedDictionaryItem(c, h.db, auth, &item) {
		return
	}
	var req models.DictionaryItem
	if !bind(c, &req) {
		return
	}
	if item.IsSystem && auth.Role != models.RoleSuperAdmin {
		fail(c, http.StatusForbidden, "家庭管理员不能修改系统预置字典")
		return
	}
	h.db.Model(&item).Updates(map[string]any{"label": req.Label, "value": req.Value, "sort_order": req.SortOrder, "enabled": req.Enabled})
	h.audit(c, item.FamilyID, &auth.UserID, "update_dictionary_item", "dictionary_item", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, item)
}

func (h *AppHandler) DeleteDictionaryItem(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.DictionaryItem
	if !scopedDictionaryItem(c, h.db, auth, &item) {
		return
	}
	if item.IsSystem && auth.Role != models.RoleSuperAdmin {
		fail(c, http.StatusForbidden, "家庭管理员不能删除系统预置字典")
		return
	}
	h.db.Delete(&item)
	h.audit(c, item.FamilyID, &auth.UserID, "delete_dictionary_item", "dictionary_item", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) ListContacts(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	q := h.db.Model(&models.Contact{})
	q = scopeFamily(q, auth, c.Query("family_id"))
	if keyword := c.Query("keyword"); keyword != "" {
		q = q.Where("name ILIKE ? OR phone ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var items []models.Contact
	page(c, q.Order("id desc"), &items)
}

func (h *AppHandler) CreateContact(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var req models.Contact
	if !bind(c, &req) {
		return
	}
	if auth.Role != models.RoleSuperAdmin {
		req.FamilyID = *auth.FamilyID
	} else if req.FamilyID == 0 {
		fail(c, http.StatusBadRequest, "super_admin 必须指定 family_id")
		return
	}
	req.CreatedBy = auth.UserID
	req.Status = defaultString(req.Status, models.StatusEnabled)
	if err := h.db.Create(&req).Error; err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &req.FamilyID, &auth.UserID, "create_contact", "contact", strconv.Itoa(int(req.ID)), true, nil)
	ok(c, req)
}

func (h *AppHandler) GetContact(c *gin.Context) {
	auth, _ := middleware.RequireFamily(c)
	var item models.Contact
	if !scopedFamilyRecord(c, h.db, auth, &item) {
		return
	}
	var pending []models.GiftRecord
	h.db.Where("family_id = ? AND contact_id = ? AND reciprocity_status = ?", item.FamilyID, item.ID, models.ReciprocityPending).Find(&pending)
	ok(c, gin.H{"contact": item, "summary": h.contactSummary(item.FamilyID, item.ID), "pending_records": pending})
}

func (h *AppHandler) UpdateContact(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.Contact
	if !scopedFamilyRecord(c, h.db, auth, &item) {
		return
	}
	var req models.Contact
	if !bind(c, &req) {
		return
	}
	h.db.Model(&item).Updates(map[string]any{"name": req.Name, "contact_type": req.ContactType, "relation_type": req.RelationType, "phone": req.Phone, "address": req.Address, "remark": req.Remark, "status": req.Status})
	h.audit(c, &item.FamilyID, &auth.UserID, "update_contact", "contact", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, item)
}

func (h *AppHandler) DeleteContact(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.Contact
	if !scopedFamilyRecord(c, h.db, auth, &item) {
		return
	}
	h.db.Delete(&item)
	h.audit(c, &item.FamilyID, &auth.UserID, "delete_contact", "contact", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) ListGiftRecords(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	q := h.db.Preload("Contact").Model(&models.GiftRecord{})
	q = scopeFamily(q, auth, c.Query("family_id"))
	if direction := c.Query("direction"); direction != "" {
		q = q.Where("direction = ?", direction)
	}
	if contactID := c.Query("contact_id"); contactID != "" {
		q = q.Where("contact_id = ?", contactID)
	}
	if occasionType := c.Query("occasion_type"); occasionType != "" {
		q = q.Where("occasion_type = ?", occasionType)
	}
	if reciprocityStatus := c.Query("reciprocity_status"); reciprocityStatus != "" {
		q = q.Where("reciprocity_status = ?", reciprocityStatus)
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("occurred_on >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("occurred_on <= ?", t)
		}
	}
	var items []models.GiftRecord
	page(c, q.Order("occurred_on desc,id desc"), &items)
}

func (h *AppHandler) CreateGiftRecord(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	req, okReq := bindGiftRecordRequest(c)
	if !okReq {
		return
	}
	record := req.toModel()
	if auth.Role != models.RoleSuperAdmin {
		record.FamilyID = *auth.FamilyID
	} else if record.FamilyID == 0 {
		fail(c, http.StatusBadRequest, "super_admin 必须指定 family_id")
		return
	}
	if !h.contactBelongsToFamily(record.ContactID, record.FamilyID) {
		fail(c, http.StatusBadRequest, "往来对象不存在或不属于当前家庭")
		return
	}
	record.CreatedBy = auth.UserID
	record.ReciprocityStatus = defaultString(record.ReciprocityStatus, models.ReciprocityNone)
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return h.syncRelatedGiftRecord(tx, &record)
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &record.FamilyID, &auth.UserID, "create_gift_record", "gift_record", strconv.Itoa(int(record.ID)), true, nil)
	ok(c, record)
}

type giftRecordRequest struct {
	FamilyID          uint   `json:"family_id"`
	ContactID         uint   `json:"contact_id" binding:"required"`
	OccasionType      string `json:"occasion_type" binding:"required"`
	Direction         string `json:"direction" binding:"required"`
	AmountCents       int64  `json:"amount_cents"`
	GiftDescription   string `json:"gift_description"`
	OccurredOn        string `json:"occurred_on" binding:"required"`
	HandledBy         string `json:"handled_by"`
	ReciprocityStatus string `json:"reciprocity_status"`
	RelatedRecordID   *uint  `json:"related_record_id"`
	Remark            string `json:"remark"`
	parsedOccurredOn  time.Time
}

func bindGiftRecordRequest(c *gin.Context) (giftRecordRequest, bool) {
	var req giftRecordRequest
	if !bind(c, &req) {
		return req, false
	}
	occurredOn, err := time.Parse("2006-01-02", req.OccurredOn)
	if err != nil {
		fail(c, http.StatusBadRequest, "发生日期必须是 yyyy-MM-dd 格式")
		return req, false
	}
	if req.Direction != models.DirectionIn && req.Direction != models.DirectionOut {
		fail(c, http.StatusBadRequest, "往来方向只能是 in 或 out")
		return req, false
	}
	req.parsedOccurredOn = occurredOn
	return req, true
}

func (req giftRecordRequest) toModel() models.GiftRecord {
	return models.GiftRecord{
		FamilyID:          req.FamilyID,
		ContactID:         req.ContactID,
		OccasionType:      req.OccasionType,
		Direction:         req.Direction,
		AmountCents:       req.AmountCents,
		GiftDescription:   req.GiftDescription,
		OccurredOn:        req.parsedOccurredOn,
		HandledBy:         req.HandledBy,
		ReciprocityStatus: req.ReciprocityStatus,
		RelatedRecordID:   req.RelatedRecordID,
		Remark:            req.Remark,
	}
}

func (h *AppHandler) GetGiftRecord(c *gin.Context) {
	auth, _ := middleware.RequireFamily(c)
	var item models.GiftRecord
	if !scopedFamilyRecord(c, h.db.Preload("Contact"), auth, &item) {
		return
	}
	ok(c, item)
}

func (h *AppHandler) UpdateGiftRecord(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.GiftRecord
	if !scopedFamilyRecord(c, h.db, auth, &item) {
		return
	}
	oldRelatedRecordID := item.RelatedRecordID
	req, okReq := bindGiftRecordRequest(c)
	if !okReq {
		return
	}
	if !h.contactBelongsToFamily(req.ContactID, item.FamilyID) {
		fail(c, http.StatusBadRequest, "往来对象不存在或不属于当前家庭")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.unlinkRelatedGiftRecord(tx, item.ID, item.FamilyID, oldRelatedRecordID, req.RelatedRecordID); err != nil {
			return err
		}
		if err := tx.Model(&item).Updates(map[string]any{"contact_id": req.ContactID, "occasion_type": req.OccasionType, "direction": req.Direction, "amount_cents": req.AmountCents, "gift_description": req.GiftDescription, "occurred_on": req.parsedOccurredOn, "handled_by": req.HandledBy, "reciprocity_status": req.ReciprocityStatus, "related_record_id": req.RelatedRecordID, "remark": req.Remark}).Error; err != nil {
			return err
		}
		item.ContactID = req.ContactID
		item.OccasionType = req.OccasionType
		item.Direction = req.Direction
		item.ReciprocityStatus = req.ReciprocityStatus
		item.RelatedRecordID = req.RelatedRecordID
		return h.syncRelatedGiftRecord(tx, &item)
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &item.FamilyID, &auth.UserID, "update_gift_record", "gift_record", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, item)
}

func (h *AppHandler) syncRelatedGiftRecord(tx *gorm.DB, record *models.GiftRecord) error {
	if record.ReciprocityStatus != models.ReciprocityDone || record.RelatedRecordID == nil {
		return nil
	}
	if *record.RelatedRecordID == record.ID {
		return errors.New("不能关联当前流水自身")
	}
	var related models.GiftRecord
	if err := tx.Where("id = ? AND family_id = ?", *record.RelatedRecordID, record.FamilyID).First(&related).Error; err != nil {
		return errors.New("关联回礼流水不存在或不属于当前家庭")
	}
	if related.ContactID != record.ContactID {
		return errors.New("关联回礼流水必须属于同一个往来对象")
	}
	if related.Direction == record.Direction {
		return errors.New("关联回礼流水方向必须与当前流水相反")
	}
	if err := tx.Model(record).Updates(map[string]any{"reciprocity_status": models.ReciprocityDone, "related_record_id": related.ID}).Error; err != nil {
		return err
	}
	return tx.Model(&related).Updates(map[string]any{"reciprocity_status": models.ReciprocityDone, "related_record_id": record.ID}).Error
}

func (h *AppHandler) unlinkRelatedGiftRecord(tx *gorm.DB, recordID uint, familyID uint, oldRelatedRecordID *uint, newRelatedRecordID *uint) error {
	if oldRelatedRecordID == nil {
		return nil
	}
	if newRelatedRecordID != nil && *newRelatedRecordID == *oldRelatedRecordID {
		return nil
	}
	return tx.Model(&models.GiftRecord{}).
		Where("id = ? AND family_id = ? AND related_record_id = ?", *oldRelatedRecordID, familyID, recordID).
		Updates(map[string]any{"related_record_id": nil, "reciprocity_status": models.ReciprocityPending}).Error
}

func (h *AppHandler) DeleteGiftRecord(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var item models.GiftRecord
	if !scopedFamilyRecord(c, h.db, auth, &item) {
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.unlinkRelatedGiftRecord(tx, item.ID, item.FamilyID, item.RelatedRecordID, nil); err != nil {
			return err
		}
		return tx.Delete(&item).Error
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &item.FamilyID, &auth.UserID, "delete_gift_record", "gift_record", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) StatsOverview(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	startDate, endDate := dateRangeFromQuery(c)
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, c.Query("family_id"))
	q = applyGiftRecordDateRange(q, startDate, endDate)
	var inAmount, outAmount int64
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&inAmount)
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&outAmount)
	var recordCount, contactCount int64
	q.Session(&gorm.Session{}).Count(&recordCount)
	q.Session(&gorm.Session{}).Distinct("contact_id").Count(&contactCount)
	trendGranularity := h.trendGranularity(auth, c.Query("family_id"), startDate, endDate)

	ok(c, gin.H{
		"cards":             gin.H{"received_cents": inAmount, "given_cents": outAmount, "net_cents": inAmount - outAmount, "contact_count": contactCount, "record_count": recordCount},
		"monthly_trend":     h.trendData(auth, c.Query("family_id"), startDate, endDate, trendGranularity),
		"trend_granularity": trendGranularity,
		"occasion_pie":      h.groupSum(auth, c.Query("family_id"), "occasion_type", startDate, endDate),
		"relation_stats":    h.relationStats(auth, c.Query("family_id"), startDate, endDate),
		"contact_top10":     h.contactTop10(auth, c.Query("family_id"), startDate, endDate),
	})
}

func (h *AppHandler) ListAuditLogs(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	q := h.db.Preload("Operator").Model(&models.AuditLog{})
	if auth.Role != models.RoleSuperAdmin {
		q = q.Where("family_id = ?", *auth.FamilyID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Joins("LEFT JOIN users audit_operator ON audit_operator.id = audit_logs.operator_id").
			Where("audit_logs.action ILIKE ? OR audit_logs.resource_type ILIKE ? OR audit_logs.ip ILIKE ? OR audit_operator.username ILIKE ? OR audit_operator.display_name ILIKE ?", like, like, like, like, like)
	}
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if success := c.Query("success"); success != "" {
		q = q.Where("success = ?", success == "true" || success == "1")
	}
	var items []models.AuditLog
	page(c, q.Order("id desc"), &items)
}

func (h *AppHandler) GetBackupConfig(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var cfg models.BackupConfig
	if err := h.db.FirstOrCreate(&cfg, models.BackupConfig{FamilyID: familyID}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "读取备份配置失败")
		return
	}
	ok(c, cfg)
}

func (h *AppHandler) UpdateBackupConfig(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var req struct {
		Enabled         bool               `json:"enabled"`
		Frequency       string             `json:"frequency" binding:"required"`
		RecipientEmails models.StringArray `json:"recipient_emails"`
		EncryptEnabled  bool               `json:"encrypt_enabled"`
		ZipPassword     string             `json:"zip_password"`
		ZipPasswordHint string             `json:"zip_password_hint"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Frequency != "daily" && req.Frequency != "weekly" && req.Frequency != "monthly" {
		fail(c, http.StatusBadRequest, "备份频率只能是 daily、weekly 或 monthly")
		return
	}
	if req.EncryptEnabled && config.AppConfig.Encryption.AppKey == "" {
		fail(c, http.StatusBadRequest, "未配置 APP_ENCRYPTION_KEY，不能开启 ZIP 加密")
		return
	}
	var cfg models.BackupConfig
	h.db.FirstOrCreate(&cfg, models.BackupConfig{FamilyID: familyID})
	if req.EncryptEnabled && req.ZipPassword == "" && cfg.ZipPasswordEncrypted == "" {
		fail(c, http.StatusBadRequest, "开启 ZIP 加密时必须设置 ZIP 密码")
		return
	}
	encrypted := ""
	if req.ZipPassword != "" {
		var err error
		encrypted, err = encryptText(req.ZipPassword, config.AppConfig.Encryption.AppKey)
		if err != nil {
			fail(c, http.StatusBadRequest, "ZIP 密码加密失败")
			return
		}
	}
	updates := map[string]any{"enabled": req.Enabled, "frequency": req.Frequency, "recipient_emails": req.RecipientEmails, "encrypt_enabled": req.EncryptEnabled, "zip_password_hint": req.ZipPasswordHint}
	if encrypted != "" {
		updates["zip_password_encrypted"] = encrypted
	}
	h.db.Model(&cfg).Updates(updates)
	auth, _ := middleware.CurrentUser(c)
	h.audit(c, &familyID, &auth.UserID, "update_backup_config", "backup_config", strconv.Itoa(int(cfg.ID)), true, models.JSONMap{"frequency": req.Frequency, "enabled": req.Enabled, "encrypt_enabled": req.EncryptEnabled})
	ok(c, cfg)
}

func (h *AppHandler) ManualBackup(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	backup, err := exporter.GenerateFamilyBackup(h.db, familyID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成备份文件失败")
		return
	}
	content, fileName, contentType, err := h.prepareBackupAttachment(familyID, backup.FileName, backup.Content)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	sum := sha256.Sum256(content)
	now := time.Now()
	job := models.BackupJob{FamilyID: familyID, TriggerType: "manual", Status: "success", FileName: fileName, FileSize: int64(len(content)), FileSHA256: fmt.Sprintf("%x", sum[:]), StartedAt: now, FinishedAt: &now, CreatedBy: &auth.UserID}
	h.db.Create(&job)
	h.audit(c, &familyID, &auth.UserID, "manual_export", "backup_job", strconv.Itoa(int(job.ID)), true, nil)
	if c.Query("email") == "true" {
		recipients := h.backupRecipients(familyID, auth.UserID)
		if err := mailer.SendBackup(recipients, "礼尚往来家庭备份", backupEmailBody("手动导出", fileName, now), mailer.Attachment{FileName: fileName, ContentType: contentType, Content: content}); err != nil {
			h.db.Model(&job).Updates(map[string]any{"status": "failed", "error_message": err.Error()})
			fail(c, http.StatusBadRequest, "备份邮件发送失败："+err.Error())
			return
		}
	}
	if c.Query("download") == "true" {
		setDownloadHeaders(c, fileName)
		c.Data(http.StatusOK, contentType, content)
		return
	}
	ok(c, job)
}

func (h *AppHandler) DownloadImportTemplate(c *gin.Context) {
	template, err := exporter.GenerateImportTemplate()
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成导入模板失败")
		return
	}
	setDownloadHeaders(c, template.FileName)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", template.Content)
}

func (h *AppHandler) ListBackupJobs(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var items []models.BackupJob
	page(c, h.db.Model(&models.BackupJob{}).Where("family_id = ?", familyID).Order("id desc"), &items)
}

func (h *AppHandler) ImportDryRun(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请上传导出的 Excel 文件，字段名为 file")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		fail(c, http.StatusBadRequest, "读取上传文件失败")
		return
	}
	result, err := importer.DryRun(h.db, familyID, content)
	if err != nil {
		fail(c, http.StatusBadRequest, "解析导入文件失败："+err.Error())
		return
	}
	payload, err := importer.PayloadToJSONMap(result.Payload)
	if err != nil {
		fail(c, http.StatusInternalServerError, "保存导入批次失败")
		return
	}
	now := time.Now()
	status := "dry_run_ready"
	if len(result.Errors) > 0 {
		status = "dry_run_failed"
	}
	job := models.ImportJob{FamilyID: familyID, OperatorID: auth.UserID, Status: status, DryRun: true, SourceFileName: header.Filename, SourcePayload: payload, CreatedCount: result.Created, UpdatedCount: result.Updated, SkippedCount: result.Skipped, ErrorCount: len(result.Errors), StartedAt: now, FinishedAt: &now}
	h.db.Create(&job)
	for _, rowErr := range result.Errors {
		h.db.Create(&models.ImportJobError{ImportJobID: job.ID, SheetName: rowErr.SheetName, RowNumber: rowErr.RowNumber, FieldName: rowErr.FieldName, ErrorMessage: rowErr.ErrorMessage, RawData: mapToJSONMap(rowErr.RawData)})
	}
	h.audit(c, &familyID, &auth.UserID, "import_dry_run", "import_job", strconv.Itoa(int(job.ID)), true, nil)
	ok(c, gin.H{"job": job, "errors": result.Errors, "source_sheets": result.SourceSheets})
}

func (h *AppHandler) ListImportJobs(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var items []models.ImportJob
	page(c, h.db.Model(&models.ImportJob{}).Where("family_id = ?", familyID).Order("id desc"), &items)
}

func (h *AppHandler) ListImportErrors(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var job models.ImportJob
	if !findByID(c, h.db, &job) {
		return
	}
	if auth.Role != models.RoleSuperAdmin && (auth.FamilyID == nil || job.FamilyID != *auth.FamilyID) {
		fail(c, http.StatusForbidden, "无权查看该导入批次")
		return
	}
	var items []models.ImportJobError
	page(c, h.db.Model(&models.ImportJobError{}).Where("import_job_id = ?", job.ID).Order("id asc"), &items)
}

func (h *AppHandler) ExecuteImport(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var job models.ImportJob
	if !findByID(c, h.db, &job) {
		return
	}
	if auth.Role != models.RoleSuperAdmin && (auth.FamilyID == nil || job.FamilyID != *auth.FamilyID) {
		fail(c, http.StatusForbidden, "无权执行该导入批次")
		return
	}
	if job.ErrorCount > 0 {
		fail(c, http.StatusBadRequest, "导入校验存在错误，不能正式导入")
		return
	}
	payload, err := importer.JSONMapToPayload(job.SourcePayload)
	if err != nil {
		fail(c, http.StatusBadRequest, "导入批次数据损坏")
		return
	}
	backup, err := exporter.GenerateFamilyBackup(h.db, job.FamilyID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "导入前备份生成失败")
		return
	}
	content, fileName, contentType, err := h.prepareBackupAttachment(job.FamilyID, backup.FileName, backup.Content)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	recipients := h.backupRecipients(job.FamilyID, auth.UserID)
	if err := mailer.SendBackup(recipients, "导入执行前自动备份", backupEmailBody("导入执行前自动备份", fileName, time.Now()), mailer.Attachment{FileName: fileName, ContentType: contentType, Content: content}); err != nil {
		h.db.Model(&job).Updates(map[string]any{"status": "blocked", "error_message": "导入前备份邮件发送失败：" + err.Error()})
		fail(c, http.StatusBadRequest, "导入前备份邮件发送失败，已中止导入："+err.Error())
		return
	}
	result, err := importer.Apply(h.db, job.FamilyID, payload, auth.UserID)
	now := time.Now()
	updates := map[string]any{"dry_run": false, "finished_at": now}
	if err != nil {
		updates["status"] = "failed"
		updates["error_message"] = err.Error()
		updates["error_count"] = len(result.Errors)
	} else {
		updates["status"] = "success"
		updates["created_count"] = result.Created
		updates["updated_count"] = result.Updated
		updates["skipped_count"] = result.Skipped
		updates["error_count"] = len(result.Errors)
	}
	h.db.Model(&job).Updates(updates)
	h.audit(c, &job.FamilyID, &auth.UserID, "execute_import", "import_job", strconv.Itoa(int(job.ID)), err == nil, models.JSONMap{"created": result.Created, "updated": result.Updated, "skipped": result.Skipped, "errors": len(result.Errors)})
	if err != nil {
		fail(c, http.StatusBadRequest, "执行导入失败："+err.Error())
		return
	}
	ok(c, gin.H{"job": job, "result": result})
}

func seedDictionary(tx *gorm.DB) error {
	types := []models.DictionaryType{
		{Code: "relation_type", Name: "关系类型", AllowFamilyCustom: true, Enabled: true, SortOrder: 10},
		{Code: "contact_type", Name: "对象类型", AllowFamilyCustom: true, Enabled: true, SortOrder: 20},
		{Code: "occasion_type", Name: "事项类型", AllowFamilyCustom: true, Enabled: true, SortOrder: 30},
		{Code: "gift_type", Name: "礼品类型", AllowFamilyCustom: true, Enabled: true, SortOrder: 40},
		{Code: "payment_method", Name: "支付方式", AllowFamilyCustom: true, Enabled: true, SortOrder: 50},
	}
	for _, t := range types {
		if err := tx.Where("code = ?", t.Code).FirstOrCreate(&t).Error; err != nil {
			return err
		}
		defaults := map[string][]string{
			"relation_type":  {"亲戚", "朋友", "同事", "同学", "邻居"},
			"contact_type":   {"个人", "家庭", "单位"},
			"occasion_type":  {"婚礼", "满月", "生日", "升学", "乔迁", "探病"},
			"gift_type":      {"礼金", "礼品"},
			"payment_method": {"现金", "微信", "支付宝", "银行转账"},
		}
		for idx, label := range defaults[t.Code] {
			item := models.DictionaryItem{TypeID: t.ID, Label: label, Value: label, SortOrder: idx + 1, Enabled: true, IsSystem: true}
			if err := tx.Where("type_id = ? AND family_id IS NULL AND value = ?", t.ID, label).FirstOrCreate(&item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *AppHandler) audit(c *gin.Context, familyID *uint, operatorID *uint, action, resourceType, resourceID string, success bool, detail models.JSONMap) {
	if detail == nil {
		detail = models.JSONMap{}
	}
	_ = h.db.Create(&models.AuditLog{FamilyID: familyID, OperatorID: operatorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Success: success, IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), Detail: detail}).Error
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

func (h *AppHandler) contactBelongsToFamily(contactID uint, familyID uint) bool {
	var count int64
	h.db.Model(&models.Contact{}).Where("id = ? AND family_id = ?", contactID, familyID).Count(&count)
	return count > 0
}

func (h *AppHandler) contactSummary(familyID, contactID uint) gin.H {
	var inAmount, outAmount int64
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&inAmount)
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&outAmount)
	return gin.H{"received_cents": inAmount, "given_cents": outAmount, "net_cents": inAmount - outAmount}
}

func dateRangeFromQuery(c *gin.Context) (*time.Time, *time.Time) {
	var startDate, endDate *time.Time
	if value := c.Query("start_date"); value != "" {
		if t, err := time.Parse("2006-01-02", value); err == nil {
			startDate = &t
		}
	}
	if value := c.Query("end_date"); value != "" {
		if t, err := time.Parse("2006-01-02", value); err == nil {
			endDate = &t
		}
	}
	return startDate, endDate
}

func applyGiftRecordDateRange(q *gorm.DB, startDate *time.Time, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		q = q.Where("occurred_on >= ?", *startDate)
	}
	if endDate != nil {
		q = q.Where("occurred_on <= ?", *endDate)
	}
	return q
}

func (h *AppHandler) trendGranularity(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) string {
	rangeStart, rangeEnd := startDate, endDate
	if rangeStart == nil || rangeEnd == nil {
		var bounds struct {
			MinDate *time.Time
			MaxDate *time.Time
		}
		q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
		q = applyGiftRecordDateRange(q, startDate, endDate)
		q.Select("MIN(occurred_on) AS min_date, MAX(occurred_on) AS max_date").Scan(&bounds)
		if rangeStart == nil {
			rangeStart = bounds.MinDate
		}
		if rangeEnd == nil {
			rangeEnd = bounds.MaxDate
		}
	}
	if rangeStart != nil && rangeEnd != nil && rangeEnd.Sub(*rangeStart).Hours()/24 <= 90 {
		return "day"
	}
	return "month"
}

func (h *AppHandler) trendData(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, granularity string) []gin.H {
	var rows []struct {
		Period    string
		Direction string
		Amount    int64
	}
	dateFormat := "YYYY-MM"
	if granularity == "day" {
		dateFormat = "YYYY-MM-DD"
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Select("TO_CHAR(occurred_on, ?) AS period, direction, COALESCE(SUM(amount_cents),0) AS amount", dateFormat).
		Group("period,direction").Order("period asc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"period": row.Period, "month": row.Period, "direction": row.Direction, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) groupSum(auth middleware.AuthUser, familyID, field string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Select(field + " AS name, COALESCE(SUM(amount_cents),0) AS amount").
		Group(field).Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) relationStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.relation_type AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount").
		Group("contacts.relation_type").Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) contactTop10(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.name AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount").
		Group("contacts.name").Order("amount desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) targetFamilyID(c *gin.Context) (uint, bool) {
	auth, _ := middleware.CurrentUser(c)
	if auth.Role == models.RoleSuperAdmin {
		id, err := strconv.Atoi(c.DefaultQuery("family_id", "0"))
		if err != nil || id <= 0 {
			fail(c, http.StatusBadRequest, "super_admin 必须指定 family_id")
			return 0, false
		}
		return uint(id), true
	}
	if auth.FamilyID == nil {
		fail(c, http.StatusForbidden, "当前账号未归属家庭")
		return 0, false
	}
	return *auth.FamilyID, true
}

func (h *AppHandler) backupRecipients(familyID uint, operatorID uint) []string {
	recipients := []string{}
	var cfg models.BackupConfig
	if err := h.db.Where("family_id = ?", familyID).First(&cfg).Error; err == nil {
		recipients = append(recipients, cfg.RecipientEmails...)
	}
	var user models.User
	if err := h.db.First(&user, operatorID).Error; err == nil && user.Email != "" {
		recipients = append(recipients, user.Email)
	}
	return recipients
}

func (h *AppHandler) prepareBackupAttachment(familyID uint, fileName string, content []byte) ([]byte, string, string, error) {
	var cfg models.BackupConfig
	if err := h.db.Where("family_id = ?", familyID).First(&cfg).Error; err != nil || !cfg.EncryptEnabled {
		return content, fileName, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	}
	password, err := decryptText(cfg.ZipPasswordEncrypted, config.AppConfig.Encryption.AppKey)
	if err != nil || password == "" {
		return nil, "", "", errors.New("备份 ZIP 加密已启用，但密码不可用")
	}
	zipContent, zipName, err := exporter.EncryptZip(fileName, content, password)
	if err != nil {
		return nil, "", "", err
	}
	return zipContent, zipName, "application/zip", nil
}

func setDownloadHeaders(c *gin.Context, fileName string) {
	escaped := strings.ReplaceAll(fileName, `"`, "")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, escaped, urlQueryEscape(fileName)))
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer("+", "%20")
	return replacer.Replace(url.QueryEscape(value))
}

func backupEmailBody(scene string, fileName string, sentAt time.Time) string {
	return fmt.Sprintf("您好：\n\n本邮件由礼尚往来系统自动发送。\n\n备份场景：%s\n附件名称：%s\n导出时间：%s\n\n附件包含当前家庭的往来对象、往来流水、家庭字典、家庭成员和统计摘要。若附件为 ZIP 文件，说明当前家庭已开启备份加密，请使用备份配置中的密码提示确认解压密码。\n\n如您未进行相关操作，请及时联系系统管理员。", scene, fileName, sentAt.Format("2006-01-02 15:04:05"))
}

func mapToJSONMap(value map[string]string) models.JSONMap {
	out := models.JSONMap{}
	for key, item := range value {
		out[key] = item
	}
	return out
}

func scopeFamily(q *gorm.DB, auth middleware.AuthUser, familyID string) *gorm.DB {
	if auth.Role == models.RoleSuperAdmin && familyID != "" {
		return q.Where("family_id = ?", familyID)
	}
	if auth.Role != models.RoleSuperAdmin && auth.FamilyID != nil {
		return q.Where("family_id = ?", *auth.FamilyID)
	}
	return q
}

func scopeGiftRecords(q *gorm.DB, auth middleware.AuthUser, familyID string) *gorm.DB {
	if auth.Role == models.RoleSuperAdmin && familyID != "" {
		return q.Where("gift_records.family_id = ?", familyID)
	}
	if auth.Role != models.RoleSuperAdmin && auth.FamilyID != nil {
		return q.Where("gift_records.family_id = ?", *auth.FamilyID)
	}
	return q
}

func scopedFamilyRecord[T any](c *gin.Context, db *gorm.DB, auth middleware.AuthUser, dest *T) bool {
	id := c.Param("id")
	q := db.Where("id = ?", id)
	if auth.Role != models.RoleSuperAdmin && auth.FamilyID != nil {
		q = q.Where("family_id = ?", *auth.FamilyID)
	}
	if err := q.First(dest).Error; err != nil {
		fail(c, http.StatusNotFound, "资源不存在")
		return false
	}
	return true
}

func scopedUser(c *gin.Context, db *gorm.DB, auth middleware.AuthUser, user *models.User) bool {
	q := db.Where("id = ?", c.Param("id"))
	if auth.Role != models.RoleSuperAdmin {
		q = q.Where("family_id = ?", *auth.FamilyID).Where("role <> ?", models.RoleSuperAdmin)
	}
	if err := q.First(user).Error; err != nil {
		fail(c, http.StatusNotFound, "用户不存在")
		return false
	}
	return true
}

func scopedDictionaryItem(c *gin.Context, db *gorm.DB, auth middleware.AuthUser, item *models.DictionaryItem) bool {
	q := db.Where("id = ?", c.Param("id"))
	if auth.Role != models.RoleSuperAdmin {
		q = q.Where("family_id = ?", *auth.FamilyID)
	}
	if err := q.First(item).Error; err != nil {
		fail(c, http.StatusNotFound, "字典项不存在")
		return false
	}
	return true
}

func findByID[T any](c *gin.Context, db *gorm.DB, dest *T) bool {
	if err := db.First(dest, c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, "资源不存在")
		return false
	}
	return true
}

func page[T any](c *gin.Context, q *gorm.DB, out *[]T) {
	pageNum, pageSize := paginationParams(c)
	var total int64
	q.Count(&total)
	q.Offset((pageNum - 1) * pageSize).Limit(pageSize).Find(out)
	ok(c, gin.H{"total": total, "list": out, "page": pageNum, "page_size": pageSize})
}

func paginationParams(c *gin.Context) (int, int) {
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(config.AppConfig.Server.Page.DefaultPageNum)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(config.AppConfig.Server.Page.DefaultPageSize)))
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > config.AppConfig.Server.Page.MaxPageSize {
		pageSize = config.AppConfig.Server.Page.MaxPageSize
	}
	return pageNum, pageSize
}

func sanitizeUser(user models.User) gin.H {
	return gin.H{"id": user.ID, "family_id": user.FamilyID, "family": user.Family, "username": user.Username, "display_name": user.DisplayName, "email": user.Email, "role": user.Role, "must_change_password": user.MustChangePassword, "status": user.Status, "last_login_at": user.LastLoginAt, "created_at": user.CreatedAt}
}

func fillFamilyAdmins(db *gorm.DB, families []models.Family) {
	adminIDs := []uint{}
	for _, family := range families {
		if family.AdminUserID != nil {
			adminIDs = append(adminIDs, *family.AdminUserID)
		}
	}
	if len(adminIDs) == 0 {
		return
	}
	var admins []models.User
	db.Where("id IN ?", adminIDs).Find(&admins)
	adminMap := map[uint]*models.User{}
	for i := range admins {
		adminMap[admins[i].ID] = &admins[i]
	}
	for i := range families {
		if families[i].AdminUserID != nil {
			families[i].AdminUser = adminMap[*families[i].AdminUserID]
		}
	}
}

func bind(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		fail(c, http.StatusBadRequest, validationMessage(err, out))
		return false
	}
	return true
}

func validationMessage(err error, out any) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err.Error()
	}
	if len(validationErrors) == 0 {
		return "请求参数不合法"
	}
	field := jsonFieldName(out, validationErrors[0].StructNamespace())
	switch validationErrors[0].Tag() {
	case "required":
		return field + "为必填项"
	case "min":
		return field + "长度不能少于 " + validationErrors[0].Param() + " 位"
	default:
		return field + "格式不正确"
	}
}

func validatePassword(password string) string {
	if len([]rune(password)) < 10 {
		return "密码长度不能少于 10 位"
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsSpace(r):
			return "密码不能包含空白字符"
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	missing := []string{}
	if !hasUpper {
		missing = append(missing, "大写字母")
	}
	if !hasLower {
		missing = append(missing, "小写字母")
	}
	if !hasDigit {
		missing = append(missing, "数字")
	}
	if !hasSpecial {
		missing = append(missing, "特殊字符")
	}
	if len(missing) > 0 {
		return "密码必须包含" + strings.Join(missing, "、")
	}
	return ""
}

func jsonFieldName(out any, namespace string) string {
	parts := strings.Split(namespace, ".")
	t := reflect.TypeOf(out)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := "字段"
	for _, part := range parts[1:] {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			break
		}
		field, ok := t.FieldByName(part)
		if !ok {
			return part
		}
		fieldName := strings.Split(field.Tag.Get("json"), ",")[0]
		if fieldName != "" && fieldName != "-" {
			name = fieldName
		} else {
			name = part
		}
		t = field.Type
	}
	return name
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success", "data": data})
}

func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"code": status, "msg": msg, "data": nil})
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func encryptText(text, key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	sum := make([]byte, 32)
	copy(sum, []byte(key))
	block, err := aes.NewCipher(sum)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decryptText(text, key string) (string, error) {
	if key == "" {
		return "", errors.New("empty key")
	}
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	sum := make([]byte, 32)
	copy(sum, []byte(key))
	block, err := aes.NewCipher(sum)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid ciphertext")
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
