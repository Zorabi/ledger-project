package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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
	if !h.userBelongsToFamily(req.OwnerUserID, req.FamilyID) {
		fail(c, http.StatusBadRequest, "归属成员不属于当前家庭")
		return
	}
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
	if !scopedFamilyRecord(c, h.db.Preload("OwnerUser"), auth, &item) {
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
	if !h.userBelongsToFamily(req.OwnerUserID, item.FamilyID) {
		fail(c, http.StatusBadRequest, "归属成员不属于当前家庭")
		return
	}
	h.db.Model(&item).Updates(map[string]any{"owner_user_id": req.OwnerUserID, "name": req.Name, "contact_type": req.ContactType, "relation_type": req.RelationType, "phone": req.Phone, "address": req.Address, "remark": req.Remark, "status": req.Status})
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

func (h *AppHandler) contactSummary(familyID, contactID uint) gin.H {
	var inAmount, outAmount int64
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&inAmount)
	h.db.Model(&models.GiftRecord{}).Where("family_id = ? AND contact_id = ? AND direction = ?", familyID, contactID, models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&outAmount)
	return gin.H{"received_cents": inAmount, "given_cents": outAmount, "net_cents": inAmount - outAmount}
}
