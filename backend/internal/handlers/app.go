package handlers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ledger-backend/internal/config"
	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
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
	protected.GET("/imports/manual/template", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.DownloadManualImportTemplate)
	protected.POST("/imports/manual/dry-run", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.ManualImportDryRun)
	protected.POST("/imports/backup/dry-run", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.BackupImportDryRun)
	protected.GET("/imports/template", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.DownloadManualImportTemplate)
	protected.POST("/imports/dry-run", middleware.RequireRoles(models.RoleSuperAdmin, models.RoleFamilyAdmin), h.BackupImportDryRun)
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
