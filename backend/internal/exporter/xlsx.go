package exporter

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	azip "github.com/alexmullins/zip"
	"gorm.io/gorm"

	"ledger-backend/internal/models"
)

type FamilyBackup struct {
	FileName string
	Content  []byte
}

func GenerateFamilyBackup(db *gorm.DB, familyID uint) (*FamilyBackup, error) {
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
		FileName: fmt.Sprintf("family-%d-backup-%s.xlsx", familyID, time.Now().Format("20060102150405")),
		Content:  content,
	}, nil
}

type sheet struct {
	name string
	rows [][]string
}

func buildXLSX(sheets []sheet) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	files := map[string]string{
		"[Content_Types].xml":        contentTypes(len(sheets)),
		"_rels/.rels":                rootRels(),
		"xl/workbook.xml":            workbookXML(sheets),
		"xl/_rels/workbook.xml.rels": workbookRels(len(sheets)),
		"xl/styles.xml":              stylesXML(),
	}
	for name, body := range files {
		if err := addZipFile(zw, name, []byte(body)); err != nil {
			return nil, err
		}
	}
	for i, s := range sheets {
		if err := addZipFile(zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), []byte(sheetXML(s.rows))); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func addZipFile(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

func contentTypes(count int) string {
	var overrides bytes.Buffer
	for i := 1; i <= count; i++ {
		overrides.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i))
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
		overrides.String() + `</Types>`
}

func rootRels() string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`
}

func workbookXML(sheets []sheet) string {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for i, s := range sheets {
		b.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(s.name), i+1, i+1))
	}
	b.WriteString(`</sheets></workbook>`)
	return b.String()
}

func workbookRels(count int) string {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for i := 1; i <= count; i++ {
		b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i, i))
	}
	b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, count+1))
	b.WriteString(`</Relationships>`)
	return b.String()
}

func stylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Arial"/></font></fonts><fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="1"><xf xfId="0"/></cellXfs></styleSheet>`
}

func sheetXML(rows [][]string) string {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for r, row := range rows {
		b.WriteString(fmt.Sprintf(`<row r="%d">`, r+1))
		for c, value := range row {
			ref := cellRef(c+1, r+1)
			b.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, xmlEscape(value)))
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func cellRef(col, row int) string {
	name := ""
	for col > 0 {
		col--
		name = string(rune('A'+col%26)) + name
		col /= 26
	}
	return name + strconv.Itoa(row)
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func giftRecordRows(records []models.GiftRecord) [][]string {
	rows := [][]string{{"业务编号", "对象编号", "对象名称", "事项类型", "方向", "金额(分)", "礼品说明", "发生日期", "经办人", "回礼状态", "关联记录编号", "备注"}}
	for _, r := range records {
		related := ""
		if r.RelatedRecordID != nil {
			related = strconv.Itoa(int(*r.RelatedRecordID))
		}
		rows = append(rows, []string{itoa(r.ID), itoa(r.ContactID), r.Contact.Name, r.OccasionType, r.Direction, strconv.FormatInt(r.AmountCents, 10), r.GiftDescription, r.OccurredOn.Format("2006-01-02"), r.HandledBy, r.ReciprocityStatus, related, r.Remark})
	}
	return rows
}

func contactRows(contacts []models.Contact) [][]string {
	rows := [][]string{{"对象编号", "名称", "对象类型", "关系类型", "手机号", "地址", "状态", "备注"}}
	for _, c := range contacts {
		rows = append(rows, []string{itoa(c.ID), c.Name, c.ContactType, c.RelationType, c.Phone, c.Address, c.Status, c.Remark})
	}
	return rows
}

func dictionaryRows(items []models.DictionaryItem) [][]string {
	rows := [][]string{{"字典编号", "类型编码", "类型名称", "标签", "值", "排序", "启用"}}
	for _, i := range items {
		rows = append(rows, []string{itoa(i.ID), i.Type.Code, i.Type.Name, i.Label, i.Value, strconv.Itoa(i.SortOrder), strconv.FormatBool(i.Enabled)})
	}
	return rows
}

func userRows(users []models.User) [][]string {
	rows := [][]string{{"用户编号", "用户名", "显示名", "邮箱", "角色", "状态", "必须改密"}}
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
		{"累计收礼(分)", strconv.FormatInt(inAmount, 10)},
		{"累计送礼(分)", strconv.FormatInt(outAmount, 10)},
		{"净差额(分)", strconv.FormatInt(inAmount-outAmount, 10)},
		{"往来对象数", strconv.Itoa(len(contacts))},
		{"记录总数", strconv.Itoa(len(records))},
	}
}

func itoa(id uint) string {
	return strconv.Itoa(int(id))
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
