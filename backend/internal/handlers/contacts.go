package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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
