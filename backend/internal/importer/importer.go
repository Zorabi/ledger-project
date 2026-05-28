package importer

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"ledger-backend/internal/models"
)

type Payload struct {
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

func Apply(db *gorm.DB, familyID uint, payload Payload, operatorID uint) (*Result, error) {
	result := &Result{Payload: payload}
	err := db.Transaction(func(tx *gorm.DB) error {
		applyContacts(tx, familyID, payload.Contacts, operatorID, result)
		applyDictionaries(tx, familyID, payload.Dictionaries, result)
		applyUsers(tx, familyID, payload.Users, result)
		applyRecords(tx, familyID, payload.Records, operatorID, result)
		if len(result.Errors) > 0 {
			return errors.New("导入数据仍存在错误")
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
	return out, json.Unmarshal(b, &out)
}

func ParseBackup(content []byte) (Payload, []string, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return Payload{}, nil, err
	}
	sheets := map[string][][]string{}
	names := []string{"往来流水", "往来对象", "家庭字典", "家庭成员"}
	for i, name := range names {
		rows, err := readSheet(zr, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1))
		if err != nil {
			return Payload{}, nil, err
		}
		sheets[name] = rows
	}
	return Payload{
		Records:      rowsToMaps(sheets["往来流水"]),
		Contacts:     rowsToMaps(sheets["往来对象"]),
		Dictionaries: rowsToMaps(sheets["家庭字典"]),
		Users:        rowsToMaps(sheets["家庭成员"]),
	}, names, nil
}

func readSheet(zr *zip.Reader, name string) ([][]string, error) {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		return parseSheetXML(data), nil
	}
	return nil, fmt.Errorf("缺少工作表 %s", name)
}

func parseSheetXML(data []byte) [][]string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rows := [][]string{}
	current := []string{}
	inT := false
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "row" {
				current = []string{}
			}
			if t.Name.Local == "t" {
				inT = true
			}
		case xml.EndElement:
			if t.Name.Local == "row" {
				rows = append(rows, current)
			}
			if t.Name.Local == "t" {
				inT = false
			}
		case xml.CharData:
			if inT {
				current = append(current, string(t))
			}
		}
	}
	return rows
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
				item[header] = strings.TrimSpace(row[i])
			}
		}
		out = append(out, item)
	}
	return out
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
		if row["对象编号"] == "" || row["方向"] == "" || row["发生日期"] == "" {
			addError(result, "往来流水", i+2, "必填字段", "对象编号、方向、发生日期不能为空", row)
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
			Status:       fallback(row["状态"], models.StatusEnabled),
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
		enabled := row["启用"] != "false"
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
		updates := map[string]any{"display_name": row["显示名"], "email": row["邮箱"], "status": fallback(row["状态"], models.StatusEnabled)}
		if err := tx.Model(&models.User{}).Where("id = ? AND family_id = ?", id, familyID).Updates(updates).Error; err != nil {
			addError(result, "家庭成员", i+2, "用户编号", err.Error(), row)
			continue
		}
		result.Updated++
	}
}

func applyRecords(tx *gorm.DB, familyID uint, rows []map[string]string, operatorID uint, result *Result) {
	for i, row := range rows {
		contactID := parseID(row["对象编号"])
		if contactID == 0 || !exists(tx, &models.Contact{}, familyID, contactID) {
			addError(result, "往来流水", i+2, "对象编号", "对象不存在", row)
			continue
		}
		occurredOn, err := time.Parse("2006-01-02", row["发生日期"])
		if err != nil {
			addError(result, "往来流水", i+2, "发生日期", "日期格式必须为 YYYY-MM-DD", row)
			continue
		}
		amount, _ := strconv.ParseInt(row["金额(分)"], 10, 64)
		related := parseID(row["关联记录编号"])
		var relatedPtr *uint
		if related > 0 {
			relatedPtr = &related
		}
		record := models.GiftRecord{
			FamilyID:          familyID,
			ContactID:         contactID,
			OccasionType:      fallback(row["事项类型"], "其他"),
			Direction:         row["方向"],
			AmountCents:       amount,
			GiftDescription:   row["礼品说明"],
			OccurredOn:        occurredOn,
			HandledBy:         row["经办人"],
			ReciprocityStatus: fallback(row["回礼状态"], models.ReciprocityNone),
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
