package importer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ledger-backend/internal/models"
)

type Payload struct {
	Mode         string              `json:"mode"`
	Contacts     []map[string]string `json:"contacts"`
	Records      []map[string]string `json:"records"`
	Dictionaries []map[string]string `json:"dictionaries"`
	Users        []map[string]string `json:"users"`
}

type Result struct {
	Payload      Payload
	Created      int
	Updated      int
	Skipped      int
	Errors       []RowError
	Conflicts    int
	SourceSheets []string
}

type RowError struct {
	SheetName    string            `json:"sheet_name"`
	RowNumber    int               `json:"row_number"`
	FieldName    string            `json:"field_name"`
	ErrorMessage string            `json:"error_message"`
	RawData      map[string]string `json:"raw_data"`
}

func DryRun(db *gorm.DB, familyID uint, content []byte) (*Result, error) {
	return DryRunBackup(db, familyID, content)
}

func DryRunBackup(db *gorm.DB, familyID uint, content []byte) (*Result, error) {
	payload, sheets, err := ParseBackup(content)
	if err != nil {
		return nil, err
	}
	result := &Result{Payload: payload, SourceSheets: sheets}
	inspectContacts(db, familyID, payload.Contacts, result)
	inspectDictionaries(db, familyID, payload.Dictionaries, result)
	inspectUsers(db, familyID, payload.Users, result)
	inspectRecords(db, familyID, payload.Records, result)
	return result, nil
}

func DryRunManual(db *gorm.DB, familyID uint, content []byte) (*Result, error) {
	payload, sheets, err := ParseManual(content)
	if err != nil {
		return nil, err
	}
	result := &Result{Payload: payload, SourceSheets: sheets}
	inspectContacts(db, familyID, payload.Contacts, result)
	inspectRecords(db, familyID, payload.Records, result)
	return result, nil
}

func Apply(db *gorm.DB, familyID uint, payload Payload, operatorID uint) (*Result, error) {
	if payload.Mode == "manual" {
		return ApplyManual(db, familyID, payload, operatorID)
	}
	return ApplyBackup(db, familyID, payload, operatorID)
}

func ApplyBackup(db *gorm.DB, familyID uint, payload Payload, operatorID uint) (*Result, error) {
	result := &Result{Payload: payload}
	err := db.Transaction(func(tx *gorm.DB) error {
		applyContacts(tx, familyID, payload.Contacts, operatorID, result)
		applyDictionaries(tx, familyID, payload.Dictionaries, result)
		applyUsers(tx, familyID, payload.Users, result)
		applyRecords(tx, familyID, payload.Records, operatorID, result)
		if len(result.Errors) > 0 {
			return errors.New(result.Errors[0].ErrorMessage)
		}
		return nil
	})
	return result, err
}

func ApplyManual(db *gorm.DB, familyID uint, payload Payload, operatorID uint) (*Result, error) {
	result := &Result{Payload: payload}
	err := db.Transaction(func(tx *gorm.DB) error {
		applyContacts(tx, familyID, payload.Contacts, operatorID, result)
		applyRecords(tx, familyID, payload.Records, operatorID, result)
		if len(result.Errors) > 0 {
			return errors.New(result.Errors[0].ErrorMessage)
		}
		return nil
	})
	return result, err
}

func PayloadToJSONMap(payload Payload) (models.JSONMap, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var out models.JSONMap
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func JSONMapToPayload(value models.JSONMap) (Payload, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return Payload{}, err
	}
	var out Payload
	if err := json.Unmarshal(b, &out); err != nil {
		return Payload{}, err
	}
	normalizePayloadRows(&out)
	return out, nil
}

func ParseBackup(content []byte) (Payload, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return Payload{}, nil, err
	}
	defer f.Close()
	sheets := map[string][][]string{}
	names := []string{"往来流水", "往来对象", "家庭字典", "家庭成员"}
	for _, name := range names {
		rows, err := readSheet(f, name)
		if err != nil {
			return Payload{}, nil, err
		}
		sheets[name] = rows
	}
	return Payload{
		Mode:         "backup",
		Records:      rowsToMaps(sheets["往来流水"]),
		Contacts:     rowsToMaps(sheets["往来对象"]),
		Dictionaries: rowsToMaps(sheets["家庭字典"]),
		Users:        rowsToMaps(sheets["家庭成员"]),
	}, names, nil
}

func ParseManual(content []byte) (Payload, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return Payload{}, nil, err
	}
	defer f.Close()
	sourceSheets := []string{}
	records, err := readSheet(f, "往来流水")
	if err == nil {
		sourceSheets = append(sourceSheets, "往来流水")
	}
	contacts, err := readSheet(f, "往来对象")
	if err == nil {
		sourceSheets = append(sourceSheets, "往来对象")
	}
	if len(sourceSheets) == 0 {
		return Payload{}, nil, errors.New("缺少工作表 往来流水 或 往来对象")
	}
	return Payload{
		Mode:     "manual",
		Records:  rowsToMaps(records),
		Contacts: rowsToMaps(contacts),
	}, sourceSheets, nil
}

func readSheet(f *excelize.File, name string) ([][]string, error) {
	index, err := f.GetSheetIndex(name)
	if err != nil || index == -1 {
		return nil, fmt.Errorf("缺少工作表 %s", name)
	}
	return f.GetRows(name)
}

func rowsToMaps(rows [][]string) []map[string]string {
	if len(rows) < 2 {
		return nil
	}
	headers := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		item := map[string]string{}
		for i, header := range headers {
			if i < len(row) {
				item[normalizeHeader(header)] = strings.TrimSpace(row[i])
			}
		}
		out = append(out, item)
	}
	return out
}

func normalizePayloadRows(payload *Payload) {
	for _, rows := range [][]map[string]string{payload.Records, payload.Contacts, payload.Dictionaries, payload.Users} {
		for _, row := range rows {
			for key, value := range row {
				normalized := normalizeHeader(key)
				if normalized != key {
					if _, exists := row[normalized]; !exists {
						row[normalized] = value
					}
					delete(row, key)
				}
			}
		}
	}
}

func normalizeHeader(header string) string {
	header = strings.TrimSpace(header)
	aliases := map[string]string{
		"往来方向": "方向",
		"方向":   "方向",
		"金额":   "金额(元)",
		"日期":   "发生日期",
	}
	if value, ok := aliases[header]; ok {
		return value
	}
	return header
}

func normalizeExcelDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return value
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006/01/02 15:04:05", "2006/01/02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	// Excel serial date system starts at 1899-12-30 when accounting for the
	// historical leap-year bug. Fractional days are intentionally discarded.
	date := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(serial))
	return date.Format("2006-01-02")
}

func inspectContacts(db *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		if row["名称"] == "" {
			addError(result, "往来对象", i+2, "名称", "名称不能为空", row)
			continue
		}
		if id := parseID(row["对象编号"]); id > 0 && exists(db, &models.Contact{}, familyID, id) {
			result.Updated++
		} else {
			result.Created++
		}
	}
}

func inspectDictionaries(db *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		if row["类型编码"] == "" || row["标签"] == "" {
			addError(result, "家庭字典", i+2, "标签", "类型编码和标签不能为空", row)
			continue
		}
		if id := parseID(row["字典编号"]); id > 0 && exists(db, &models.DictionaryItem{}, familyID, id) {
			result.Updated++
		} else {
			result.Created++
		}
	}
}

func inspectUsers(db *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		if row["用户名"] == "" {
			addError(result, "家庭成员", i+2, "用户名", "用户名不能为空", row)
			continue
		}
		if id := parseID(row["用户编号"]); id > 0 && exists(db, &models.User{}, familyID, id) {
			result.Updated++
		} else {
			result.Skipped++
		}
	}
}

func inspectRecords(db *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		row["发生日期"] = normalizeExcelDate(row["发生日期"])
		if row["方向"] == "" || row["发生日期"] == "" {
			addError(result, "往来流水", i+2, "必填字段", "方向、发生日期不能为空", row)
			continue
		}
		if normalizeDirection(row["方向"]) == "" {
			addError(result, "往来流水", i+2, "往来方向", "往来方向必须填写收礼或送礼", row)
			continue
		}
		if row["对象编号"] == "" && row["对象名称"] == "" {
			addError(result, "往来流水", i+2, "往来对象", "对象编号和对象名称至少填写一个", row)
			continue
		}
		if _, err := parseAmountCents(row); err != nil {
			addError(result, "往来流水", i+2, "金额", err.Error(), row)
			continue
		}
		if id := parseID(row["业务编号"]); id > 0 && exists(db, &models.GiftRecord{}, familyID, id) {
			result.Updated++
		} else {
			result.Created++
		}
	}
}

func applyContacts(tx *gorm.DB, familyID uint, rows []map[string]string, operatorID uint, result *Result) {
	for i, row := range rows {
		if row["名称"] == "" {
			addError(result, "往来对象", i+2, "名称", "名称不能为空", row)
			continue
		}
		id := parseID(row["对象编号"])
		contact := models.Contact{
			FamilyID:     familyID,
			Name:         row["名称"],
			ContactType:  fallback(row["对象类型"], "个人"),
			RelationType: fallback(row["关系类型"], "其他"),
			Phone:        row["手机号"],
			Address:      row["地址"],
			Status:       normalizeStatus(fallback(row["状态"], models.StatusEnabled)),
			Remark:       row["备注"],
			CreatedBy:    operatorID,
		}
		if id > 0 && exists(tx, &models.Contact{}, familyID, id) {
			tx.Model(&models.Contact{}).Where("id = ? AND family_id = ?", id, familyID).Updates(contact)
			result.Updated++
		} else {
			tx.Create(&contact)
			result.Created++
		}
	}
}

func applyDictionaries(tx *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		if row["类型编码"] == "" || row["标签"] == "" {
			addError(result, "家庭字典", i+2, "标签", "类型编码和标签不能为空", row)
			continue
		}
		var dtype models.DictionaryType
		if err := tx.Where("code = ?", row["类型编码"]).First(&dtype).Error; err != nil {
			addError(result, "家庭字典", i+2, "类型编码", "字典类型不存在", row)
			continue
		}
		enabled := normalizeBool(row["启用"])
		item := models.DictionaryItem{TypeID: dtype.ID, FamilyID: &familyID, Label: row["标签"], Value: fallback(row["值"], row["标签"]), SortOrder: int(parseID(row["排序"])), Enabled: enabled, IsSystem: false}
		id := parseID(row["字典编号"])
		if id > 0 && exists(tx, &models.DictionaryItem{}, familyID, id) {
			tx.Model(&models.DictionaryItem{}).Where("id = ? AND family_id = ?", id, familyID).Updates(item)
			result.Updated++
		} else {
			tx.Create(&item)
			result.Created++
		}
	}
}

func applyUsers(tx *gorm.DB, familyID uint, rows []map[string]string, result *Result) {
	for i, row := range rows {
		id := parseID(row["用户编号"])
		if id == 0 {
			result.Skipped++
			continue
		}
		if row["用户名"] == "" {
			addError(result, "家庭成员", i+2, "用户名", "用户名不能为空", row)
			continue
		}
		updates := map[string]any{"display_name": row["显示名"], "email": row["邮箱"], "status": normalizeStatus(fallback(row["状态"], models.StatusEnabled))}
		if err := tx.Model(&models.User{}).Where("id = ? AND family_id = ?", id, familyID).Updates(updates).Error; err != nil {
			addError(result, "家庭成员", i+2, "用户编号", err.Error(), row)
			continue
		}
		result.Updated++
	}
}

func applyRecords(tx *gorm.DB, familyID uint, rows []map[string]string, operatorID uint, result *Result) {
	for i, row := range rows {
		row["发生日期"] = normalizeExcelDate(row["发生日期"])
		contactID, err := resolveContactID(tx, familyID, row, operatorID)
		if err != nil {
			addError(result, "往来流水", i+2, "往来对象", err.Error(), row)
			continue
		}
		occurredOn, err := time.Parse("2006-01-02", row["发生日期"])
		if err != nil {
			addError(result, "往来流水", i+2, "发生日期", "日期格式必须为 YYYY-MM-DD", row)
			continue
		}
		amount, err := parseAmountCents(row)
		if err != nil {
			addError(result, "往来流水", i+2, "金额", err.Error(), row)
			continue
		}
		direction := normalizeDirection(row["方向"])
		if direction == "" {
			addError(result, "往来流水", i+2, "往来方向", "往来方向必须填写收礼或送礼", row)
			continue
		}
		related := parseID(row["关联记录编号"])
		var relatedPtr *uint
		if related > 0 {
			relatedPtr = &related
		}
		record := models.GiftRecord{
			FamilyID:          familyID,
			ContactID:         contactID,
			OccasionType:      fallback(row["事项类型"], "其他"),
			Direction:         direction,
			AmountCents:       amount,
			GiftDescription:   row["礼品说明"],
			OccurredOn:        occurredOn,
			HandledBy:         row["经办人"],
			ReciprocityStatus: normalizeReciprocityStatus(fallback(row["回礼状态"], models.ReciprocityNone)),
			RelatedRecordID:   relatedPtr,
			Remark:            row["备注"],
			CreatedBy:         operatorID,
		}
		id := parseID(row["业务编号"])
		if id > 0 && exists(tx, &models.GiftRecord{}, familyID, id) {
			tx.Model(&models.GiftRecord{}).Where("id = ? AND family_id = ?", id, familyID).Updates(record)
			result.Updated++
		} else {
			tx.Create(&record)
			result.Created++
		}
	}
}

func exists[T any](db *gorm.DB, model *T, familyID uint, id uint) bool {
	var count int64
	db.Model(model).Where("id = ? AND family_id = ?", id, familyID).Count(&count)
	return count > 0
}

func contactNameExists(db *gorm.DB, familyID uint, name string) bool {
	var count int64
	db.Model(&models.Contact{}).Where("family_id = ? AND name = ?", familyID, strings.TrimSpace(name)).Count(&count)
	return count > 0
}

func resolveContactID(tx *gorm.DB, familyID uint, row map[string]string, operatorID uint) (uint, error) {
	if id := parseID(row["对象编号"]); id > 0 {
		if exists(tx, &models.Contact{}, familyID, id) {
			return id, nil
		}
		if strings.TrimSpace(row["对象名称"]) == "" {
			return 0, errors.New("对象编号不存在，请填写对象名称以便自动创建")
		}
	}
	name := strings.TrimSpace(row["对象名称"])
	if name == "" {
		return 0, errors.New("对象编号和对象名称至少填写一个")
	}
	var contact models.Contact
	err := tx.Where("family_id = ? AND name = ?", familyID, name).Limit(1).Find(&contact).Error
	if err != nil {
		return 0, err
	}
	if contact.ID > 0 {
		return contact.ID, nil
	}
	contact = models.Contact{
		FamilyID:     familyID,
		Name:         name,
		ContactType:  "个人",
		RelationType: "其他",
		Status:       models.StatusEnabled,
		CreatedBy:    operatorID,
	}
	if err := tx.Create(&contact).Error; err != nil {
		return 0, err
	}
	return contact.ID, nil
}

func addError(result *Result, sheet string, row int, field string, message string, raw map[string]string) {
	result.Errors = append(result.Errors, RowError{SheetName: sheet, RowNumber: row, FieldName: field, ErrorMessage: message, RawData: raw})
}

func parseID(value string) uint {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	if n < 0 {
		return 0
	}
	return uint(n)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

func normalizeDirection(value string) string {
	switch strings.TrimSpace(value) {
	case models.DirectionIn, "收礼", "我家收礼", "收入":
		return models.DirectionIn
	case models.DirectionOut, "送礼", "我家送礼", "支出":
		return models.DirectionOut
	default:
		return ""
	}
}

func normalizeStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "", "启用", "正常", models.StatusEnabled:
		return models.StatusEnabled
	case "禁用", "停用", models.StatusDisabled:
		return models.StatusDisabled
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeBool(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "true", "TRUE", "是", "启用", "开启":
		return true
	case "false", "FALSE", "否", "禁用", "关闭":
		return false
	default:
		return true
	}
}

func normalizeReciprocityStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "", "无需回礼", models.ReciprocityNone:
		return models.ReciprocityNone
	case "待回礼", models.ReciprocityPending:
		return models.ReciprocityPending
	case "已回礼", models.ReciprocityDone:
		return models.ReciprocityDone
	default:
		return strings.TrimSpace(value)
	}
}

func parseAmountCents(row map[string]string) (int64, error) {
	if cents := strings.TrimSpace(row["金额(分)"]); cents != "" {
		value, err := strconv.ParseInt(cents, 10, 64)
		if err != nil {
			return 0, errors.New("金额(分)必须是整数")
		}
		return value, nil
	}
	yuan := strings.TrimSpace(row["金额(元)"])
	if yuan == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(yuan, 64)
	if err != nil {
		return 0, errors.New("金额(元)必须是数字")
	}
	return int64(math.Round(value * 100)), nil
}
