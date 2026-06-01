package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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
