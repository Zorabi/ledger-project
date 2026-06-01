package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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
	var dtype models.DictionaryType
	if err := h.db.First(&dtype, req.TypeID).Error; err != nil {
		fail(c, http.StatusBadRequest, "字典类型不存在")
		return
	}
	if auth.Role == models.RoleSuperAdmin {
		req.IsSystem = req.FamilyID == nil
	} else {
		req.FamilyID = auth.FamilyID
		req.IsSystem = false
	}
	if !h.canWriteDictionaryItem(c, auth, dtype, req.FamilyID) {
		return
	}
	if !h.ensureDictionaryValueUnique(c, req.TypeID, req.FamilyID, req.Value, 0) {
		return
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
	var dtype models.DictionaryType
	if err := h.db.First(&dtype, item.TypeID).Error; err != nil {
		fail(c, http.StatusBadRequest, "字典类型不存在")
		return
	}
	if !h.canWriteDictionaryItem(c, auth, dtype, item.FamilyID) {
		return
	}
	if !h.ensureDictionaryValueUnique(c, item.TypeID, item.FamilyID, req.Value, item.ID) {
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
	var dtype models.DictionaryType
	if err := h.db.First(&dtype, item.TypeID).Error; err != nil {
		fail(c, http.StatusBadRequest, "字典类型不存在")
		return
	}
	if !h.canWriteDictionaryItem(c, auth, dtype, item.FamilyID) {
		return
	}
	h.db.Delete(&item)
	h.audit(c, item.FamilyID, &auth.UserID, "delete_dictionary_item", "dictionary_item", strconv.Itoa(int(item.ID)), true, nil)
	ok(c, nil)
}

func (h *AppHandler) canWriteDictionaryItem(c *gin.Context, auth middleware.AuthUser, dtype models.DictionaryType, familyID *uint) bool {
	if familyID == nil {
		if auth.Role != models.RoleSuperAdmin {
			fail(c, http.StatusForbidden, "只有系统管理员可以维护系统预置字典")
			return false
		}
		return true
	}
	if !dtype.AllowFamilyCustom {
		fail(c, http.StatusForbidden, "该字典类型已关闭家庭自定义，不能维护家庭字典项")
		return false
	}
	return true
}

func (h *AppHandler) ensureDictionaryValueUnique(c *gin.Context, typeID uint, familyID *uint, value string, excludeID uint) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		fail(c, http.StatusBadRequest, "字典值不能为空")
		return false
	}
	q := h.db.Model(&models.DictionaryItem{}).Where("type_id = ? AND value = ? AND enabled = true", typeID, value)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if familyID != nil {
		q = q.Where("family_id IS NULL OR family_id = ?", *familyID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		fail(c, http.StatusInternalServerError, "检查字典值唯一性失败")
		return false
	}
	if count > 0 {
		fail(c, http.StatusBadRequest, "同一字典类型下已存在相同字典值，请使用不同的值")
		return false
	}
	return true
}
