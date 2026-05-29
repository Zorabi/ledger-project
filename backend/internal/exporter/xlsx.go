package exporter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	azip "github.com/alexmullins/zip"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"ledger-backend/internal/models"
)

type FamilyBackup struct {
	FileName string
	Content  []byte
}

func GenerateFamilyBackup(db *gorm.DB, familyID uint) (*FamilyBackup, error) {
	var family models.Family
	if err := db.First(&family, familyID).Error; err != nil {
		return nil, err
	}
	var contacts []models.Contact
	var records []models.GiftRecord
	var dictionaries []models.DictionaryItem
	var users []models.User
	if err := db.Where("family_id = ?", familyID).Order("id asc").Find(&contacts).Error; err != nil {
		return nil, err
	}
	if err := db.Preload("Contact").Where("family_id = ?", familyID).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	if err := db.Preload("Type").Where("family_id = ?", familyID).Order("id asc").Find(&dictionaries).Error; err != nil {
		return nil, err
	}
	if err := db.Where("family_id = ?", familyID).Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}

	sheets := []sheet{
		{name: "往来流水", rows: giftRecordRows(records)},
		{name: "往来对象", rows: contactRows(contacts)},
		{name: "家庭字典", rows: dictionaryRows(dictionaries)},
		{name: "家庭成员", rows: userRows(users)},
		{name: "统计摘要", rows: summaryRows(records, contacts)},
	}
	content, err := buildXLSX(sheets)
	if err != nil {
		return nil, err
	}
	return &FamilyBackup{
		FileName: backupFileName(family.Name, time.Now()),
		Content:  content,
	}, nil
}

func GenerateImportTemplate() (*FamilyBackup, error) {
	sheets := []sheet{
		{name: "往来流水", rows: [][]string{giftRecordHeader()}},
		{name: "往来对象", rows: [][]string{contactHeader()}},
		{name: "家庭字典", rows: [][]string{dictionaryHeader()}},
		{name: "家庭成员", rows: [][]string{userHeader()}},
		{name: "填写说明", rows: [][]string{
			{"说明项", "内容"},
			{"导入方式", "先校验导入文件，确认无错误后再执行正式导入。"},
			{"往来对象不存在", "往来流水可填写对象名称；若对象编号为空且对象名称不存在，系统会自动创建该对象。"},
			{"金额", "推荐填写金额(元)，例如 128.50；若文件中存在金额(分)列且有值，系统会优先按整数分导入，用于兼容历史备份。"},
			{"日期", "发生日期格式为 yyyy-MM-dd。"},
		}},
	}
	content, err := buildXLSX(sheets)
	if err != nil {
		return nil, err
	}
	return &FamilyBackup{
		FileName: fmt.Sprintf("礼尚往来-导入模板-%s.xlsx", fileTime(time.Now())),
		Content:  content,
	}, nil
}

type sheet struct {
	name string
	rows [][]string
}

func buildXLSX(sheets []sheet) ([]byte, error) {
	f := excelize.NewFile()
	for i, s := range sheets {
		if i == 0 {
			_ = f.SetSheetName("Sheet1", s.name)
		} else {
			_, _ = f.NewSheet(s.name)
		}
		for r, row := range s.rows {
			cell, _ := excelize.CoordinatesToCellName(1, r+1)
			if err := f.SetSheetRow(s.name, cell, &row); err != nil {
				return nil, err
			}
		}
		if len(s.rows) > 0 {
			headerStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
			lastCell, _ := excelize.CoordinatesToCellName(len(s.rows[0]), 1)
			_ = f.SetCellStyle(s.name, "A1", lastCell, headerStyle)
			for col := 1; col <= len(s.rows[0]); col++ {
				colName, _ := excelize.ColumnNumberToName(col)
				_ = f.SetColWidth(s.name, colName, colName, 16)
			}
		}
	}
	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func giftRecordRows(records []models.GiftRecord) [][]string {
	rows := [][]string{giftRecordHeader()}
	for _, r := range records {
		related := ""
		if r.RelatedRecordID != nil {
			related = strconv.Itoa(int(*r.RelatedRecordID))
		}
		rows = append(rows, []string{itoa(r.ID), itoa(r.ContactID), r.Contact.Name, r.OccasionType, r.Direction, centsToYuan(r.AmountCents), r.GiftDescription, r.OccurredOn.Format("2006-01-02"), r.HandledBy, r.ReciprocityStatus, related, r.Remark})
	}
	return rows
}

func contactRows(contacts []models.Contact) [][]string {
	rows := [][]string{contactHeader()}
	for _, c := range contacts {
		rows = append(rows, []string{itoa(c.ID), c.Name, c.ContactType, c.RelationType, c.Phone, c.Address, c.Status, c.Remark})
	}
	return rows
}

func dictionaryRows(items []models.DictionaryItem) [][]string {
	rows := [][]string{dictionaryHeader()}
	for _, i := range items {
		rows = append(rows, []string{itoa(i.ID), i.Type.Code, i.Type.Name, i.Label, i.Value, strconv.Itoa(i.SortOrder), strconv.FormatBool(i.Enabled)})
	}
	return rows
}

func userRows(users []models.User) [][]string {
	rows := [][]string{userHeader()}
	for _, u := range users {
		rows = append(rows, []string{itoa(u.ID), u.Username, u.DisplayName, u.Email, u.Role, u.Status, strconv.FormatBool(u.MustChangePassword)})
	}
	return rows
}

func summaryRows(records []models.GiftRecord, contacts []models.Contact) [][]string {
	var inAmount, outAmount int64
	for _, r := range records {
		if r.Direction == models.DirectionIn {
			inAmount += r.AmountCents
		}
		if r.Direction == models.DirectionOut {
			outAmount += r.AmountCents
		}
	}
	return [][]string{
		{"指标", "值"},
		{"累计收礼(元)", centsToYuan(inAmount)},
		{"累计送礼(元)", centsToYuan(outAmount)},
		{"净差额(元)", centsToYuan(inAmount - outAmount)},
		{"往来对象数", strconv.Itoa(len(contacts))},
		{"记录总数", strconv.Itoa(len(records))},
	}
}

func itoa(id uint) string {
	return strconv.Itoa(int(id))
}

func giftRecordHeader() []string {
	return []string{"业务编号", "对象编号", "对象名称", "事项类型", "方向", "金额(元)", "礼品说明", "发生日期", "经办人", "回礼状态", "关联记录编号", "备注"}
}

func contactHeader() []string {
	return []string{"对象编号", "名称", "对象类型", "关系类型", "手机号", "地址", "状态", "备注"}
}

func dictionaryHeader() []string {
	return []string{"字典编号", "类型编码", "类型名称", "标签", "值", "排序", "启用"}
}

func userHeader() []string {
	return []string{"用户编号", "用户名", "显示名", "邮箱", "角色", "状态", "必须改密"}
}

func centsToYuan(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}

func backupFileName(familyName string, now time.Time) string {
	name := sanitizeFileName(familyName)
	if name == "" {
		name = "家庭"
	}
	return fmt.Sprintf("礼尚往来-%s-完整备份-%s.xlsx", name, fileTime(now))
}

func sanitizeFileName(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "'", "<", "-", ">", "-", "|", "-")
	return strings.Trim(replacer.Replace(value), ". ")
}

func fileTime(value time.Time) string {
	return value.Format("2006-01-02_15-04")
}

func EncryptZip(fileName string, content []byte, password string) ([]byte, string, error) {
	if password == "" {
		return nil, "", fmt.Errorf("ZIP 密码不能为空")
	}
	zipName := strings.TrimSuffix(fileName, ".xlsx") + ".zip"
	buf := &bytes.Buffer{}
	zw := azip.NewWriter(buf)
	w, err := zw.Encrypt(fileName, password)
	if err != nil {
		return nil, "", err
	}
	if _, err := w.Write(content); err != nil {
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), zipName, nil
}
