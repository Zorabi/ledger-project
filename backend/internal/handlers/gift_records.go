package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

func (h *AppHandler) ListGiftRecords(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	q := h.db.Preload("Contact").Model(&models.GiftRecord{})
	q = scopeGiftRecords(q, auth, c.Query("family_id"))
	if direction := c.Query("direction"); direction != "" {
		q = q.Where("direction = ?", direction)
	}
	if contactID := c.Query("contact_id"); contactID != "" {
		q = q.Where("contact_id = ?", contactID)
	}
	contactJoined := false
	joinContact := func() {
		if contactJoined {
			return
		}
		q = q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id")
		contactJoined = true
	}
	if ownerUserID := c.Query("owner_user_id"); ownerUserID != "" {
		joinContact()
		q = q.Where("contacts.owner_user_id = ?", ownerUserID)
	}
	if contactType := c.Query("contact_type"); contactType != "" {
		joinContact()
		q = q.Where("contacts.contact_type = ?", contactType)
	}
	if relationType := c.Query("relation_type"); relationType != "" {
		joinContact()
		q = q.Where("contacts.relation_type = ?", relationType)
	}
	if occasionType := c.Query("occasion_type"); occasionType != "" {
		q = q.Where("occasion_type = ?", occasionType)
	}
	if reciprocityStatus := c.Query("reciprocity_status"); reciprocityStatus != "" {
		q = q.Where("reciprocity_status = ?", reciprocityStatus)
	}
	if c.Query("reciprocity_unlinked") == "true" {
		q = q.Where("gift_records.reciprocity_status = ? AND gift_records.related_record_id IS NULL", models.ReciprocityDone)
	}
	if remark := c.Query("remark"); remark != "" {
		keyword := "%" + remark + "%"
		q = q.Where("(remark ILIKE ? OR gift_description ILIKE ?)", keyword, keyword)
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
	h.db.Preload("Contact").First(&record, record.ID)
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
	h.db.Preload("Contact").First(&item, item.ID)
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
	if err := tx.Model(&models.GiftRecord{}).
		Where("id = ? AND family_id = ?", record.ID, record.FamilyID).
		Select("reciprocity_status", "related_record_id").
		Updates(map[string]any{"reciprocity_status": models.ReciprocityDone, "related_record_id": related.ID}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.GiftRecord{}).
		Where("id = ? AND family_id = ?", related.ID, record.FamilyID).
		Select("reciprocity_status", "related_record_id").
		Updates(map[string]any{"reciprocity_status": models.ReciprocityDone, "related_record_id": record.ID}).Error; err != nil {
		return err
	}
	record.ReciprocityStatus = models.ReciprocityDone
	record.RelatedRecordID = &related.ID
	return nil
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
