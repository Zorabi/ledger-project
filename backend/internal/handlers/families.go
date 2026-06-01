package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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
