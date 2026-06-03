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
		actionCodes := csvQuery(c.Query("keyword_actions"))
		resourceCodes := csvQuery(c.Query("keyword_resources"))
		successFilter, hasSuccessFilter := boolQuery(c.Query("keyword_success"))
		q = q.Joins("LEFT JOIN users audit_operator ON audit_operator.id = audit_logs.operator_id").
			Where(
				h.auditKeywordSQL(actionCodes, resourceCodes, hasSuccessFilter),
				h.auditKeywordArgs(like, actionCodes, resourceCodes, hasSuccessFilter, successFilter)...,
			)
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

func (h *AppHandler) auditKeywordSQL(actionCodes, resourceCodes []string, hasSuccessFilter bool) string {
	parts := []string{
		"audit_logs.action ILIKE ?",
		"audit_logs.resource_type ILIKE ?",
		"audit_logs.resource_id ILIKE ?",
		"audit_logs.ip ILIKE ?",
		"audit_logs.user_agent ILIKE ?",
		"CAST(audit_logs.detail AS TEXT) ILIKE ?",
		"audit_operator.username ILIKE ?",
		"audit_operator.display_name ILIKE ?",
	}
	if len(actionCodes) > 0 {
		parts = append(parts, "audit_logs.action IN ?")
	}
	if len(resourceCodes) > 0 {
		parts = append(parts, "audit_logs.resource_type IN ?")
	}
	if hasSuccessFilter {
		parts = append(parts, "audit_logs.success = ?")
	}
	return strings.Join(parts, " OR ")
}

func (h *AppHandler) auditKeywordArgs(like string, actionCodes, resourceCodes []string, hasSuccessFilter bool, successFilter bool) []any {
	args := []any{like, like, like, like, like, like, like, like}
	if len(actionCodes) > 0 {
		args = append(args, actionCodes)
	}
	if len(resourceCodes) > 0 {
		args = append(args, resourceCodes)
	}
	if hasSuccessFilter {
		args = append(args, successFilter)
	}
	return args
}

func csvQuery(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func boolQuery(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}

func (h *AppHandler) audit(c *gin.Context, familyID *uint, operatorID *uint, action, resourceType, resourceID string, success bool, detail models.JSONMap) {
	if detail == nil {
		detail = models.JSONMap{}
	}
	_ = h.db.Create(&models.AuditLog{FamilyID: familyID, OperatorID: operatorID, Action: action, ResourceType: resourceType, ResourceID: resourceID, Success: success, IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"), Detail: detail}).Error
}
