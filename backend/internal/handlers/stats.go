package handlers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

func (h *AppHandler) StatsOverview(c *gin.Context) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	familyID := c.Query("family_id")
	startDate, endDate := dateRangeFromQuery(c)
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	var inAmount, outAmount int64
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&inAmount)
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&outAmount)
	var recordCount, contactCount int64
	q.Session(&gorm.Session{}).Count(&recordCount)
	q.Session(&gorm.Session{}).Distinct("contact_id").Count(&contactCount)
	trendGranularity := h.trendGranularity(auth, familyID, startDate, endDate)

	ok(c, gin.H{
		"cards":             gin.H{"received_cents": inAmount, "given_cents": outAmount, "net_cents": inAmount - outAmount, "contact_count": contactCount, "record_count": recordCount},
		"monthly_trend":     h.trendData(auth, familyID, startDate, endDate, trendGranularity),
		"trend_granularity": trendGranularity,
		"occasion_pie":      h.groupSum(auth, familyID, "occasion_type", startDate, endDate),
		"relation_stats":    h.relationStats(auth, familyID, startDate, endDate),
		"contact_top10":     h.contactTop10(auth, familyID, startDate, endDate),
		"reciprocity_stats": h.reciprocityStats(auth, familyID, startDate, endDate),
		"pending_top10":     h.pendingReciprocityTop10(auth, familyID, startDate, endDate),
		"balance_top10":     h.balanceTop10(auth, familyID, startDate, endDate),
		"occasion_trend":    h.occasionTrend(auth, familyID, startDate, endDate, trendGranularity),
		"occasion_matrix":   h.occasionDirectionStats(auth, familyID, startDate, endDate),
		"relation_matrix":   h.relationDirectionStats(auth, familyID, startDate, endDate),
		"period_compare":    h.periodCompare(auth, familyID, startDate, endDate),
		"handler_stats":     h.handlerStats(auth, familyID, startDate, endDate),
		"owner_stats":       h.ownerStats(auth, familyID, startDate, endDate),
		"owner_contacts":    h.ownerContactStats(auth, familyID),
	})
}

func (h *AppHandler) ReceivablesOverview(c *gin.Context) {
	h.obligationOverview(c, models.DirectionOut)
}

func (h *AppHandler) PayablesOverview(c *gin.Context) {
	h.obligationOverview(c, models.DirectionIn)
}

func (h *AppHandler) obligationOverview(c *gin.Context, direction string) {
	auth, okAuth := middleware.RequireFamily(c)
	if !okAuth {
		return
	}
	familyID := c.Query("family_id")
	startDate, endDate := dateRangeFromQuery(c)
	pageNum, pageSize := paginationParams(c)
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	var totalAmount, recordCount, contactCount int64
	q.Session(&gorm.Session{}).Select("COALESCE(SUM(gift_records.amount_cents),0)").Scan(&totalAmount)
	q.Session(&gorm.Session{}).Count(&recordCount)
	q.Session(&gorm.Session{}).Distinct("gift_records.contact_id").Count(&contactCount)
	var oldest struct {
		OccurredOn *time.Time
	}
	q.Session(&gorm.Session{}).Select("MIN(gift_records.occurred_on) AS occurred_on").Scan(&oldest)
	maxWaitDays := int64(0)
	if oldest.OccurredOn != nil {
		maxWaitDays = int64(time.Since(*oldest.OccurredOn).Hours() / 24)
	}
	ok(c, gin.H{
		"cards": gin.H{
			"amount_cents":  totalAmount,
			"record_count":  recordCount,
			"contact_count": contactCount,
			"max_wait_days": maxWaitDays,
		},
		"contact_top":    h.obligationContactTop(auth, familyID, startDate, endDate, direction),
		"aging_stats":    h.obligationAgingStats(auth, familyID, startDate, endDate, direction),
		"owner_stats":    h.obligationOwnerStats(auth, familyID, startDate, endDate, direction),
		"relation_stats": h.obligationRelationStats(auth, familyID, startDate, endDate, direction),
		"records":        h.obligationRecords(auth, familyID, startDate, endDate, direction, pageNum, pageSize),
	})
}

func (h *AppHandler) obligationBaseQuery(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string) *gorm.DB {
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	return q.Where("gift_records.direction = ? AND (gift_records.reciprocity_status = ? OR (gift_records.reciprocity_status = ? AND gift_records.related_record_id IS NULL))", direction, models.ReciprocityPending, models.ReciprocityDone)
}

func (h *AppHandler) obligationContactTop(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string) []gin.H {
	var rows []struct {
		ContactID     uint
		Name          string
		OwnerName     string
		RelationType  string
		OccasionTypes string
		Amount        int64
		Count         int64
		LastOccurred  time.Time
		FirstOccurred time.Time
	}
	ownerNameExpr := "COALESCE(NULLIF(users.display_name, ''), NULLIF(users.username, ''), '未指定')"
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Joins("LEFT JOIN users ON users.id = contacts.owner_user_id").
		Select("contacts.id AS contact_id, contacts.name AS name, " + ownerNameExpr + " AS owner_name, contacts.relation_type AS relation_type, STRING_AGG(DISTINCT gift_records.occasion_type, ',' ORDER BY gift_records.occasion_type) AS occasion_types, COALESCE(SUM(gift_records.amount_cents),0) AS amount, COUNT(*) AS count, MAX(gift_records.occurred_on) AS last_occurred, MIN(gift_records.occurred_on) AS first_occurred").
		Group("contacts.id, contacts.name, " + ownerNameExpr + ", contacts.relation_type").
		Order("amount desc, first_occurred asc").Limit(5).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{
			"contact_id":        row.ContactID,
			"name":              row.Name,
			"owner_name":        row.OwnerName,
			"relation_type":     row.RelationType,
			"occasion_types":    splitCSV(row.OccasionTypes),
			"amount_cents":      row.Amount,
			"record_count":      row.Count,
			"last_occurred_on":  row.LastOccurred.Format("2006-01-02"),
			"first_occurred_on": row.FirstOccurred.Format("2006-01-02"),
			"wait_days":         int64(time.Since(row.FirstOccurred).Hours() / 24),
		})
	}
	return out
}

func splitCSV(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (h *AppHandler) obligationAgingStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string) []gin.H {
	var rows []struct {
		Bucket string
		Amount int64
		Count  int64
	}
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	q.Select("CASE WHEN CURRENT_DATE - gift_records.occurred_on <= 30 THEN '30天内' WHEN CURRENT_DATE - gift_records.occurred_on <= 90 THEN '31-90天' WHEN CURRENT_DATE - gift_records.occurred_on <= 180 THEN '91-180天' WHEN CURRENT_DATE - gift_records.occurred_on <= 365 THEN '181-365天' ELSE '1年以上' END AS bucket, COALESCE(SUM(gift_records.amount_cents),0) AS amount, COUNT(*) AS count").
		Group("bucket").
		Order("MIN(gift_records.occurred_on) asc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"bucket": row.Bucket, "amount_cents": row.Amount, "record_count": row.Count})
	}
	return out
}

func (h *AppHandler) obligationOwnerStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string) []gin.H {
	ownerNameExpr := "COALESCE(NULLIF(users.display_name, ''), NULLIF(users.username, ''), '未指定')"
	var rows []struct {
		Name   string
		Amount int64
		Count  int64
	}
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Joins("LEFT JOIN users ON users.id = contacts.owner_user_id").
		Select(ownerNameExpr + " AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount, COUNT(*) AS count").
		Group(ownerNameExpr).Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount, "record_count": row.Count})
	}
	return out
}

func (h *AppHandler) obligationRelationStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
		Count  int64
	}
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.relation_type AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount, COUNT(*) AS count").
		Group("contacts.relation_type").Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount, "record_count": row.Count})
	}
	return out
}

func (h *AppHandler) obligationRecords(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, direction string, pageNum int, pageSize int) gin.H {
	var rows []struct {
		ID              uint
		ContactName     string
		OwnerName       string
		RelationType    string
		OccasionType    string
		AmountCents     int64
		OccurredOn      time.Time
		Reciprocity     string
		GiftDescription string
		Remark          string
	}
	var total int64
	h.obligationBaseQuery(auth, familyID, startDate, endDate, direction).Count(&total)
	ownerNameExpr := "COALESCE(NULLIF(users.display_name, ''), NULLIF(users.username, ''), '未指定')"
	q := h.obligationBaseQuery(auth, familyID, startDate, endDate, direction)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Joins("LEFT JOIN users ON users.id = contacts.owner_user_id").
		Select("gift_records.id AS id, contacts.name AS contact_name, " + ownerNameExpr + " AS owner_name, contacts.relation_type AS relation_type, gift_records.occasion_type AS occasion_type, gift_records.amount_cents AS amount_cents, gift_records.occurred_on AS occurred_on, gift_records.reciprocity_status AS reciprocity, gift_records.gift_description AS gift_description, gift_records.remark AS remark").
		Order("gift_records.occurred_on asc, gift_records.id asc").
		Offset((pageNum - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{
			"id":               row.ID,
			"contact_name":     row.ContactName,
			"owner_name":       row.OwnerName,
			"relation_type":    row.RelationType,
			"occasion_type":    row.OccasionType,
			"amount_cents":     row.AmountCents,
			"occurred_on":      row.OccurredOn.Format("2006-01-02"),
			"wait_days":        int64(time.Since(row.OccurredOn).Hours() / 24),
			"reciprocity":      row.Reciprocity,
			"gift_description": row.GiftDescription,
			"remark":           row.Remark,
		})
	}
	return gin.H{"total": total, "list": out, "page": pageNum, "page_size": pageSize}
}

func dateRangeFromQuery(c *gin.Context) (*time.Time, *time.Time) {
	var startDate, endDate *time.Time
	if value := c.Query("start_date"); value != "" {
		if t, err := time.Parse("2006-01-02", value); err == nil {
			startDate = &t
		}
	}
	if value := c.Query("end_date"); value != "" {
		if t, err := time.Parse("2006-01-02", value); err == nil {
			endDate = &t
		}
	}
	return startDate, endDate
}

func applyGiftRecordDateRange(q *gorm.DB, startDate *time.Time, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		q = q.Where("gift_records.occurred_on >= ?", *startDate)
	}
	if endDate != nil {
		q = q.Where("gift_records.occurred_on <= ?", *endDate)
	}
	return q
}

func (h *AppHandler) trendGranularity(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) string {
	rangeStart, rangeEnd := startDate, endDate
	if rangeStart == nil || rangeEnd == nil {
		var bounds struct {
			MinDate *time.Time
			MaxDate *time.Time
		}
		q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
		q = applyGiftRecordDateRange(q, startDate, endDate)
		q.Select("MIN(occurred_on) AS min_date, MAX(occurred_on) AS max_date").Scan(&bounds)
		if rangeStart == nil {
			rangeStart = bounds.MinDate
		}
		if rangeEnd == nil {
			rangeEnd = bounds.MaxDate
		}
	}
	if rangeStart != nil && rangeEnd != nil && rangeEnd.Sub(*rangeStart).Hours()/24 <= 90 {
		return "day"
	}
	return "month"
}

func (h *AppHandler) trendData(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, granularity string) []gin.H {
	var rows []struct {
		Period    string
		Direction string
		Amount    int64
	}
	dateFormat := "YYYY-MM"
	if granularity == "day" {
		dateFormat = "YYYY-MM-DD"
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Select("TO_CHAR(occurred_on, ?) AS period, direction, COALESCE(SUM(amount_cents),0) AS amount", dateFormat).
		Group("period,direction").Order("period asc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"period": row.Period, "month": row.Period, "direction": row.Direction, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) groupSum(auth middleware.AuthUser, familyID, field string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Select(field + " AS name, COALESCE(SUM(amount_cents),0) AS amount").
		Group(field).Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) relationStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.relation_type AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount").
		Group("contacts.relation_type").Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) contactTop10(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.
		Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.name AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount").
		Group("contacts.name").Order("amount desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) reciprocityStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Status string
		Amount int64
		Count  int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Select("reciprocity_status AS status, COALESCE(SUM(amount_cents),0) AS amount, COUNT(*) AS count").
		Group("reciprocity_status").Order("amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"status": row.Status, "amount_cents": row.Amount, "count": row.Count})
	}
	return out
}

func (h *AppHandler) pendingReciprocityTop10(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name   string
		Amount int64
		Count  int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Where("gift_records.reciprocity_status = ?", models.ReciprocityPending).
		Select("contacts.name AS name, COALESCE(SUM(gift_records.amount_cents),0) AS amount, COUNT(*) AS count").
		Group("contacts.name").Order("amount desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "amount_cents": row.Amount, "count": row.Count})
	}
	return out
}

func (h *AppHandler) balanceTop10(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name     string
		Received int64
		Given    int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.name AS name, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS received, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS given", models.DirectionIn, models.DirectionOut).
		Group("contacts.name").
		Order("ABS(COALESCE(SUM(CASE WHEN gift_records.direction = 'in' THEN gift_records.amount_cents ELSE 0 END),0) - COALESCE(SUM(CASE WHEN gift_records.direction = 'out' THEN gift_records.amount_cents ELSE 0 END),0)) desc").
		Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "received_cents": row.Received, "given_cents": row.Given, "net_cents": row.Received - row.Given})
	}
	return out
}

func (h *AppHandler) occasionTrend(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, granularity string) []gin.H {
	var rows []struct {
		Period string
		Name   string
		Amount int64
	}
	dateFormat := "YYYY-MM"
	if granularity == "day" {
		dateFormat = "YYYY-MM-DD"
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Select("TO_CHAR(occurred_on, ?) AS period, occasion_type AS name, COALESCE(SUM(amount_cents),0) AS amount", dateFormat).
		Group("period, occasion_type").Order("period asc, amount desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"period": row.Period, "name": row.Name, "amount_cents": row.Amount})
	}
	return out
}

func (h *AppHandler) relationDirectionStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name      string
		Received  int64
		Given     int64
		RecordNum int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Select("contacts.relation_type AS name, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS received, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS given, COUNT(*) AS record_num", models.DirectionIn, models.DirectionOut).
		Group("contacts.relation_type").
		Order("COALESCE(SUM(gift_records.amount_cents),0) desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "received_cents": row.Received, "given_cents": row.Given, "record_count": row.RecordNum})
	}
	return out
}

func (h *AppHandler) occasionDirectionStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name      string
		Received  int64
		Given     int64
		RecordNum int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Select("occasion_type AS name, COALESCE(SUM(CASE WHEN direction = ? THEN amount_cents ELSE 0 END),0) AS received, COALESCE(SUM(CASE WHEN direction = ? THEN amount_cents ELSE 0 END),0) AS given, COUNT(*) AS record_num", models.DirectionIn, models.DirectionOut).
		Group("occasion_type").
		Order("COALESCE(SUM(amount_cents),0) desc").Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "received_cents": row.Received, "given_cents": row.Given, "record_count": row.RecordNum})
	}
	return out
}

func (h *AppHandler) periodCompare(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) gin.H {
	currentStart, currentEnd := compareCurrentRange(auth, familyID, startDate, endDate, h.db)
	if currentStart == nil || currentEnd == nil {
		return gin.H{"current": gin.H{}, "previous": gin.H{}, "delta": gin.H{}, "current_range": gin.H{}, "previous_range": gin.H{}}
	}
	duration := currentEnd.Sub(*currentStart)
	previousEnd := currentStart.Add(-24 * time.Hour)
	previousStart := previousEnd.Add(-duration)
	current := h.rangeSummary(auth, familyID, currentStart, currentEnd)
	previous := h.rangeSummary(auth, familyID, &previousStart, &previousEnd)
	return gin.H{
		"current":  current,
		"previous": previous,
		"current_range": gin.H{
			"start_date": currentStart.Format("2006-01-02"),
			"end_date":   currentEnd.Format("2006-01-02"),
		},
		"previous_range": gin.H{
			"start_date": previousStart.Format("2006-01-02"),
			"end_date":   previousEnd.Format("2006-01-02"),
		},
		"delta": gin.H{
			"received_cents": current["received_cents"].(int64) - previous["received_cents"].(int64),
			"given_cents":    current["given_cents"].(int64) - previous["given_cents"].(int64),
			"net_cents":      current["net_cents"].(int64) - previous["net_cents"].(int64),
			"record_count":   current["record_count"].(int64) - previous["record_count"].(int64),
		},
	}
}

func compareCurrentRange(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time, db *gorm.DB) (*time.Time, *time.Time) {
	if startDate != nil && endDate != nil {
		return startDate, endDate
	}
	var bounds struct {
		MinDate *time.Time
		MaxDate *time.Time
	}
	q := scopeGiftRecords(db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Select("MIN(occurred_on) AS min_date, MAX(occurred_on) AS max_date").Scan(&bounds)
	if startDate != nil {
		bounds.MinDate = startDate
	}
	if endDate != nil {
		bounds.MaxDate = endDate
	}
	return bounds.MinDate, bounds.MaxDate
}

func (h *AppHandler) rangeSummary(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) gin.H {
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	var received, given, count int64
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionIn).Select("COALESCE(SUM(amount_cents),0)").Scan(&received)
	q.Session(&gorm.Session{}).Where("direction = ?", models.DirectionOut).Select("COALESCE(SUM(amount_cents),0)").Scan(&given)
	q.Session(&gorm.Session{}).Count(&count)
	return gin.H{"received_cents": received, "given_cents": given, "net_cents": received - given, "record_count": count}
}

func (h *AppHandler) handlerStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	var rows []struct {
		Name     string
		Received int64
		Given    int64
		Count    int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Select("COALESCE(NULLIF(handled_by, ''), '未填写') AS name, COALESCE(SUM(CASE WHEN direction = ? THEN amount_cents ELSE 0 END),0) AS received, COALESCE(SUM(CASE WHEN direction = ? THEN amount_cents ELSE 0 END),0) AS given, COUNT(*) AS count", models.DirectionIn, models.DirectionOut).
		Group("name").Order("COALESCE(SUM(amount_cents),0) desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "received_cents": row.Received, "given_cents": row.Given, "record_count": row.Count})
	}
	return out
}

func (h *AppHandler) ownerStats(auth middleware.AuthUser, familyID string, startDate *time.Time, endDate *time.Time) []gin.H {
	ownerNameExpr := "COALESCE(NULLIF(users.display_name, ''), NULLIF(users.username, ''), '未指定')"
	var rows []struct {
		Name     string
		Received int64
		Given    int64
		Count    int64
	}
	q := scopeGiftRecords(h.db.Model(&models.GiftRecord{}), auth, familyID)
	q = applyGiftRecordDateRange(q, startDate, endDate)
	q.Joins("JOIN contacts ON contacts.id = gift_records.contact_id").
		Joins("LEFT JOIN users ON users.id = contacts.owner_user_id").
		Select(ownerNameExpr+" AS name, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS received, COALESCE(SUM(CASE WHEN gift_records.direction = ? THEN gift_records.amount_cents ELSE 0 END),0) AS given, COUNT(*) AS count", models.DirectionIn, models.DirectionOut).
		Group(ownerNameExpr).Order("COALESCE(SUM(gift_records.amount_cents),0) desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "received_cents": row.Received, "given_cents": row.Given, "record_count": row.Count})
	}
	return out
}

func (h *AppHandler) ownerContactStats(auth middleware.AuthUser, familyID string) []gin.H {
	ownerNameExpr := "COALESCE(NULLIF(users.display_name, ''), NULLIF(users.username, ''), '未指定')"
	var rows []struct {
		Name  string
		Count int64
	}
	q := h.db.Model(&models.Contact{})
	if auth.Role == models.RoleSuperAdmin && familyID != "" {
		q = q.Where("contacts.family_id = ?", familyID)
	}
	if auth.Role != models.RoleSuperAdmin && auth.FamilyID != nil {
		q = q.Where("contacts.family_id = ?", *auth.FamilyID)
	}
	q.Joins("LEFT JOIN users ON users.id = contacts.owner_user_id").
		Select(ownerNameExpr + " AS name, COUNT(*) AS count").
		Group(ownerNameExpr).Order("count desc").Limit(10).Scan(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"name": row.Name, "count": row.Count})
	}
	return out
}
