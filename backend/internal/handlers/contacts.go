package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

var errContactNameExists = errors.New("contact name already exists")

func (h *AppHandler) ListContacts(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	q := h.db.Preload("OwnerUser").Model(&models.Contact{})
	q = scopeFamily(q, auth, c.Query("family_id"))
	if keyword := c.Query("keyword"); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name ILIKE ? OR phone ILIKE ? OR address ILIKE ? OR remark ILIKE ?", like, like, like, like)
	}
	if contactType := c.Query("contact_type"); contactType != "" {
		q = q.Where("contact_type = ?", contactType)
	}
	if relationType := c.Query("relation_type"); relationType != "" {
		q = q.Where("relation_type = ?", relationType)
	}
	if ownerUserID := c.Query("owner_user_id"); ownerUserID != "" {
		q = q.Where("owner_user_id = ?", ownerUserID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if hasOwner := c.Query("has_owner"); hasOwner != "" {
		if hasOwner == "true" {
			q = q.Where("owner_user_id IS NOT NULL")
		} else if hasOwner == "false" {
			q = q.Where("owner_user_id IS NULL")
		}
	}
	if hasPhone := c.Query("has_phone"); hasPhone != "" {
		if hasPhone == "true" {
			q = q.Where("phone <> ''")
		} else if hasPhone == "false" {
			q = q.Where("phone = ''")
		}
	}
	if startDate := c.Query("created_start"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if endDate := c.Query("created_end"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		fail(c, http.StatusBadRequest, "往来对象名称不能为空")
		return
	}
	if !h.userBelongsToFamily(req.OwnerUserID, req.FamilyID) {
		fail(c, http.StatusBadRequest, "归属成员不属于当前家庭")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		normalizedName := normalizeContactName(req.Name)
		if err := lockContactName(tx, req.FamilyID, normalizedName); err != nil {
			return err
		}
		if contactNameTaken(tx, req.FamilyID, normalizedName, 0) {
			return errContactNameExists
		}
		return tx.Create(&req).Error
	})
	if err != nil {
		if errors.Is(err, errContactNameExists) {
			fail(c, http.StatusBadRequest, "同家庭下已存在相同名称的往来对象")
			return
		}
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	h.audit(c, &req.FamilyID, &auth.UserID, "create_contact", "contact", strconv.Itoa(int(req.ID)), true, nil)
	ok(c, req)
}

func (h *AppHandler) GetContact(c *gin.Context) {
	auth, _ := middleware.RequireFamily(c)
	var item models.Contact
	if !scopedFamilyRecord(c, h.db.Preload("OwnerUser"), auth, &item) {
		return
	}
	var todoRecords []models.GiftRecord
	h.db.
		Where("family_id = ? AND contact_id = ? AND (reciprocity_status = ? OR (reciprocity_status = ? AND related_record_id IS NULL))", item.FamilyID, item.ID, models.ReciprocityPending, models.ReciprocityDone).
		Order("occurred_on asc, amount_cents desc, id asc").
		Limit(20).
		Find(&todoRecords)
	var timeline []models.GiftRecord
	h.db.
		Where("family_id = ? AND contact_id = ?", item.FamilyID, item.ID).
		Order("occurred_on desc, id desc").
		Limit(30).
		Find(&timeline)
	var linked []models.GiftRecord
	h.db.
		Where("family_id = ? AND contact_id = ? AND reciprocity_status = ? AND related_record_id IS NOT NULL", item.FamilyID, item.ID, models.ReciprocityDone).
		Order("occurred_on desc, id desc").
		Limit(20).
		Find(&linked)
	ok(c, gin.H{
		"contact":         item,
		"summary":         h.contactSummary(item.FamilyID, item.ID),
		"pending_records": todoRecords,
		"todo_records":    todoRecords,
		"timeline":        timeline,
		"linked_records":  linked,
	})
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
	if !h.userBelongsToFamily(req.OwnerUserID, item.FamilyID) {
		fail(c, http.StatusBadRequest, "归属成员不属于当前家庭")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		fail(c, http.StatusBadRequest, "往来对象名称不能为空")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		normalizedName := normalizeContactName(req.Name)
		if err := lockContactName(tx, item.FamilyID, normalizedName); err != nil {
			return err
		}
		if contactNameTaken(tx, item.FamilyID, normalizedName, item.ID) {
			return errContactNameExists
		}
		return tx.Model(&item).Updates(map[string]any{"owner_user_id": req.OwnerUserID, "name": req.Name, "contact_type": req.ContactType, "relation_type": req.RelationType, "phone": req.Phone, "address": req.Address, "remark": req.Remark, "status": req.Status}).Error
	})
	if err != nil {
		if errors.Is(err, errContactNameExists) {
			fail(c, http.StatusBadRequest, "同家庭下已存在相同名称的往来对象")
			return
		}
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
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

func (h *AppHandler) contactBelongsToFamily(contactID uint, familyID uint) bool {
	var count int64
	h.db.Model(&models.Contact{}).Where("id = ? AND family_id = ?", contactID, familyID).Count(&count)
	return count > 0
}

func (h *AppHandler) userBelongsToFamily(userID *uint, familyID uint) bool {
	if userID == nil || *userID == 0 {
		return true
	}
	var count int64
	h.db.Model(&models.User{}).Where("id = ? AND family_id = ?", *userID, familyID).Count(&count)
	return count > 0
}

func normalizeContactName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func lockContactName(db *gorm.DB, familyID uint, normalizedName string) error {
	lockKey := strconv.Itoa(int(familyID)) + ":" + normalizedName
	return db.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error
}

func contactNameTaken(db *gorm.DB, familyID uint, normalizedName string, excludeID uint) bool {
	var count int64
	q := db.Model(&models.Contact{}).Where("family_id = ? AND LOWER(BTRIM(name)) = ?", familyID, normalizedName)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

func (h *AppHandler) contactSummary(familyID, contactID uint) gin.H {
	var inAmount, outAmount int64
	var recordCount int64
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&inAmount)
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&outAmount)
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ?", familyID, contactID).Count(&recordCount)
	return gin.H{"received_cents": inAmount, "given_cents": outAmount, "net_cents": inAmount - outAmount, "record_count": recordCount}
}
