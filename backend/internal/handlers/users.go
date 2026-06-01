package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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

func sanitizeUser(user models.User) gin.H {
	return gin.H{"id": user.ID, "family_id": user.FamilyID, "family": user.Family, "username": user.Username, "display_name": user.DisplayName, "email": user.Email, "role": user.Role, "must_change_password": user.MustChangePassword, "status": user.Status, "last_login_at": user.LastLoginAt, "created_at": user.CreatedAt}
}
