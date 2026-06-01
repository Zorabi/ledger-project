package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

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

func (h *AppHandler) audit(c *gin.Context, familyID *uint, operatorID *uint, action, resourceType, resourceID string, success bool, detail models.JSONMap) {
	if detail == nil {
		detail = models.JSONMap{}
	}
	_ = h.db.Create(&models.AuditLog{FamilyID: familyID, OperatorID: operatorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Success: success, IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), Detail: detail}).Error
}
