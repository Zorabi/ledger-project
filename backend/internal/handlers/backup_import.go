package handlers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/config"
	"ledger-backend/internal/exporter"
	"ledger-backend/internal/importer"
	"ledger-backend/internal/mailer"
	"ledger-backend/internal/middleware"
	"ledger-backend/internal/models"
)

func (h *AppHandler) GetBackupConfig(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var cfg models.BackupConfig
	if err := h.db.FirstOrCreate(&cfg, models.BackupConfig{FamilyID: familyID}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "读取备份配置失败")
		return
	}
	ok(c, cfg)
}

func (h *AppHandler) UpdateBackupConfig(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var req struct {
		Enabled         bool               `json:"enabled"`
		Frequency       string             `json:"frequency" binding:"required"`
		RecipientEmails models.StringArray `json:"recipient_emails"`
		EncryptEnabled  bool               `json:"encrypt_enabled"`
		ZipPassword     string             `json:"zip_password"`
		ZipPasswordHint string             `json:"zip_password_hint"`
	}
	if !bind(c, &req) {
		return
	}
	if req.Frequency != "daily" && req.Frequency != "weekly" && req.Frequency != "monthly" {
		fail(c, http.StatusBadRequest, "备份频率只能是 daily、weekly 或 monthly")
		return
	}
	if req.EncryptEnabled && config.AppConfig.Encryption.AppKey == "" {
		fail(c, http.StatusBadRequest, "未配置 APP_ENCRYPTION_KEY，不能开启 ZIP 加密")
		return
	}
	var cfg models.BackupConfig
	h.db.FirstOrCreate(&cfg, models.BackupConfig{FamilyID: familyID})
	if req.EncryptEnabled && req.ZipPassword == "" && cfg.ZipPasswordEncrypted == "" {
		fail(c, http.StatusBadRequest, "开启 ZIP 加密时必须设置 ZIP 密码")
		return
	}
	encrypted := ""
	if req.ZipPassword != "" {
		var err error
		encrypted, err = encryptText(req.ZipPassword, config.AppConfig.Encryption.AppKey)
		if err != nil {
			fail(c, http.StatusBadRequest, "ZIP 密码加密失败")
			return
		}
	}
	updates := map[string]any{"enabled": req.Enabled, "frequency": req.Frequency, "recipient_emails": req.RecipientEmails, "encrypt_enabled": req.EncryptEnabled, "zip_password_hint": req.ZipPasswordHint}
	if encrypted != "" {
		updates["zip_password_encrypted"] = encrypted
	}
	h.db.Model(&cfg).Updates(updates)
	auth, _ := middleware.CurrentUser(c)
	h.audit(c, &familyID, &auth.UserID, "update_backup_config", "backup_config", strconv.Itoa(int(cfg.ID)), true, models.JSONMap{"frequency": req.Frequency, "enabled": req.Enabled, "encrypt_enabled": req.EncryptEnabled})
	ok(c, cfg)
}

func (h *AppHandler) ManualBackup(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	backup, err := exporter.GenerateFamilyBackup(h.db, familyID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成备份文件失败")
		return
	}
	content, fileName, contentType, err := h.prepareBackupAttachment(familyID, backup.FileName, backup.Content)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	sum := sha256.Sum256(content)
	now := time.Now()
	job := models.BackupJob{FamilyID: familyID, TriggerType: "manual", Status: "success", FileName: fileName, FileSize: int64(len(content)), FileSHA256: fmt.Sprintf("%x", sum[:]), StartedAt: now, FinishedAt: &now, CreatedBy: &auth.UserID}
	h.db.Create(&job)
	h.audit(c, &familyID, &auth.UserID, "manual_export", "backup_job", strconv.Itoa(int(job.ID)), true, nil)
	if c.Query("email") == "true" {
		recipients := h.backupRecipients(familyID, auth.UserID)
		if err := mailer.SendBackup(recipients, "礼尚往来家庭备份", backupEmailBody("手动导出", fileName, now), mailer.Attachment{FileName: fileName, ContentType: contentType, Content: content}); err != nil {
			h.db.Model(&job).Updates(map[string]any{"status": "failed", "error_message": err.Error()})
			fail(c, http.StatusBadRequest, "备份邮件发送失败："+err.Error())
			return
		}
	}
	if c.Query("download") == "true" {
		setDownloadHeaders(c, fileName)
		c.Data(http.StatusOK, contentType, content)
		return
	}
	ok(c, job)
}

func (h *AppHandler) DownloadManualImportTemplate(c *gin.Context) {
	template, err := exporter.GenerateImportTemplate()
	if err != nil {
		fail(c, http.StatusInternalServerError, "生成导入模板失败")
		return
	}
	setDownloadHeaders(c, template.FileName)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", template.Content)
}

func (h *AppHandler) ListBackupJobs(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var items []models.BackupJob
	page(c, h.db.Model(&models.BackupJob{}).Where("family_id = ?", familyID).Order("id desc"), &items)
}

func (h *AppHandler) ImportDryRun(c *gin.Context) {
	h.runImportDryRun(c, "backup")
}

func (h *AppHandler) BackupImportDryRun(c *gin.Context) {
	h.runImportDryRun(c, "backup")
}

func (h *AppHandler) ManualImportDryRun(c *gin.Context) {
	h.runImportDryRun(c, "manual")
}

func (h *AppHandler) runImportDryRun(c *gin.Context, mode string) {
	auth, _ := middleware.CurrentUser(c)
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请上传 Excel 文件，字段名为 file")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		fail(c, http.StatusBadRequest, "读取上传文件失败")
		return
	}
	var result *importer.Result
	if mode == "manual" {
		result, err = importer.DryRunManual(h.db, familyID, content)
	} else {
		result, err = importer.DryRunBackup(h.db, familyID, content)
	}
	if err != nil {
		fail(c, http.StatusBadRequest, "解析导入文件失败："+err.Error())
		return
	}
	payload, err := importer.PayloadToJSONMap(result.Payload)
	if err != nil {
		fail(c, http.StatusInternalServerError, "保存导入批次失败")
		return
	}
	now := time.Now()
	status := "dry_run_ready"
	errorMessage := ""
	if len(result.Errors) > 0 {
		status = "dry_run_failed"
		errorMessage = importFailureMessage(result, errors.New("导入校验存在错误"))
	}
	job := models.ImportJob{FamilyID: familyID, OperatorID: auth.UserID, Status: status, DryRun: true, SourceFileName: header.Filename, SourcePayload: payload, CreatedCount: result.Created, UpdatedCount: result.Updated, SkippedCount: result.Skipped, ErrorCount: len(result.Errors), StartedAt: now, FinishedAt: &now, ErrorMessage: errorMessage}
	h.db.Create(&job)
	for _, rowErr := range result.Errors {
		h.db.Create(&models.ImportJobError{ImportJobID: job.ID, SheetName: rowErr.SheetName, RowNumber: rowErr.RowNumber, FieldName: rowErr.FieldName, ErrorMessage: rowErr.ErrorMessage, RawData: mapToJSONMap(rowErr.RawData)})
	}
	action := "import_dry_run"
	if mode == "manual" {
		action = "manual_import_dry_run"
	}
	h.audit(c, &familyID, &auth.UserID, action, "import_job", strconv.Itoa(int(job.ID)), true, nil)
	ok(c, gin.H{"job": job, "errors": result.Errors, "source_sheets": result.SourceSheets})
}

func (h *AppHandler) ListImportJobs(c *gin.Context) {
	familyID, okFamily := h.targetFamilyID(c)
	if !okFamily {
		return
	}
	var items []models.ImportJob
	page(c, h.db.Model(&models.ImportJob{}).Where("family_id = ?", familyID).Order("id desc"), &items)
}

func (h *AppHandler) ListImportErrors(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var job models.ImportJob
	if !findByID(c, h.db, &job) {
		return
	}
	if auth.Role != models.RoleSuperAdmin && (auth.FamilyID == nil || job.FamilyID != *auth.FamilyID) {
		fail(c, http.StatusForbidden, "无权查看该导入批次")
		return
	}
	var items []models.ImportJobError
	page(c, h.db.Model(&models.ImportJobError{}).Where("import_job_id = ?", job.ID).Order("id asc"), &items)
}

func (h *AppHandler) ExecuteImport(c *gin.Context) {
	auth, _ := middleware.CurrentUser(c)
	var job models.ImportJob
	if !findByID(c, h.db, &job) {
		return
	}
	if auth.Role != models.RoleSuperAdmin && (auth.FamilyID == nil || job.FamilyID != *auth.FamilyID) {
		fail(c, http.StatusForbidden, "无权执行该导入批次")
		return
	}
	if job.ErrorCount > 0 {
		fail(c, http.StatusBadRequest, "导入校验存在错误，不能正式导入")
		return
	}
	payload, err := importer.JSONMapToPayload(job.SourcePayload)
	if err != nil {
		fail(c, http.StatusBadRequest, "导入批次数据损坏")
		return
	}
	backup, err := exporter.GenerateFamilyBackup(h.db, job.FamilyID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "导入前备份生成失败")
		return
	}
	content, fileName, contentType, err := h.prepareBackupAttachment(job.FamilyID, backup.FileName, backup.Content)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	recipients := h.backupRecipients(job.FamilyID, auth.UserID)
	if err := mailer.SendBackup(recipients, "导入执行前自动备份", backupEmailBody("导入执行前自动备份", fileName, time.Now()), mailer.Attachment{FileName: fileName, ContentType: contentType, Content: content}); err != nil {
		h.db.Model(&job).Updates(map[string]any{"status": "blocked", "error_message": "导入前备份邮件发送失败：" + err.Error()})
		fail(c, http.StatusBadRequest, "导入前备份邮件发送失败，已中止导入："+err.Error())
		return
	}
	result, err := importer.Apply(h.db, job.FamilyID, payload, auth.UserID)
	now := time.Now()
	updates := map[string]any{"dry_run": false, "finished_at": now}
	if err != nil {
		updates["status"] = "failed"
		updates["error_message"] = importFailureMessage(result, err)
		updates["error_count"] = len(result.Errors)
	} else {
		updates["status"] = "success"
		updates["created_count"] = result.Created
		updates["updated_count"] = result.Updated
		updates["skipped_count"] = result.Skipped
		updates["error_count"] = len(result.Errors)
	}
	h.db.Model(&job).Updates(updates)
	if len(result.Errors) > 0 {
		h.db.Where("import_job_id = ?", job.ID).Delete(&models.ImportJobError{})
		for _, rowErr := range result.Errors {
			h.db.Create(&models.ImportJobError{ImportJobID: job.ID, SheetName: rowErr.SheetName, RowNumber: rowErr.RowNumber, FieldName: rowErr.FieldName, ErrorMessage: rowErr.ErrorMessage, RawData: mapToJSONMap(rowErr.RawData)})
		}
	}
	h.audit(c, &job.FamilyID, &auth.UserID, "execute_import", "import_job", strconv.Itoa(int(job.ID)), err == nil, models.JSONMap{"created": result.Created, "updated": result.Updated, "skipped": result.Skipped, "errors": len(result.Errors)})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": importFailureMessage(result, err), "data": gin.H{"job_id": job.ID, "errors": result.Errors}})
		return
	}
	ok(c, gin.H{"job": job, "result": result})
}

func importFailureMessage(result *importer.Result, err error) string {
	if result != nil && len(result.Errors) > 0 {
		rowErr := result.Errors[0]
		return fmt.Sprintf("第 %d 行「%s」：%s", rowErr.RowNumber, rowErr.FieldName, rowErr.ErrorMessage)
	}
	return "执行导入失败：" + err.Error()
}

func (h *AppHandler) targetFamilyID(c *gin.Context) (uint, bool) {
	auth, _ := middleware.CurrentUser(c)
	if auth.Role == models.RoleSuperAdmin {
		id, err := strconv.Atoi(c.DefaultQuery("family_id", "0"))
		if err != nil || id <= 0 {
			fail(c, http.StatusBadRequest, "super_admin 必须指定 family_id")
			return 0, false
		}
		return uint(id), true
	}
	if auth.FamilyID == nil {
		fail(c, http.StatusForbidden, "当前账号未归属家庭")
		return 0, false
	}
	return *auth.FamilyID, true
}

func (h *AppHandler) backupRecipients(familyID uint, operatorID uint) []string {
	recipients := []string{}
	var cfg models.BackupConfig
	if err := h.db.Where("family_id = ?", familyID).First(&cfg).Error; err == nil {
		recipients = append(recipients, cfg.RecipientEmails...)
	}
	var user models.User
	if err := h.db.First(&user, operatorID).Error; err == nil && user.Email != "" {
		recipients = append(recipients, user.Email)
	}
	return recipients
}

func (h *AppHandler) prepareBackupAttachment(familyID uint, fileName string, content []byte) ([]byte, string, string, error) {
	var cfg models.BackupConfig
	if err := h.db.Where("family_id = ?", familyID).First(&cfg).Error; err != nil || !cfg.EncryptEnabled {
		return content, fileName, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	}
	password, err := decryptText(cfg.ZipPasswordEncrypted, config.AppConfig.Encryption.AppKey)
	if err != nil || password == "" {
		return nil, "", "", errors.New("备份 ZIP 加密已启用，但密码不可用")
	}
	zipContent, zipName, err := exporter.EncryptZip(fileName, content, password)
	if err != nil {
		return nil, "", "", err
	}
	return zipContent, zipName, "application/zip", nil
}

func setDownloadHeaders(c *gin.Context, fileName string) {
	escaped := strings.ReplaceAll(fileName, `"`, "")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, escaped, urlQueryEscape(fileName)))
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer("+", "%20")
	return replacer.Replace(url.QueryEscape(value))
}

func backupEmailBody(scene string, fileName string, sentAt time.Time) string {
	return fmt.Sprintf("您好：\n\n本邮件由礼尚往来系统自动发送。\n\n备份场景：%s\n附件名称：%s\n导出时间：%s\n\n附件包含当前家庭的往来对象、往来流水、家庭字典、家庭成员和统计摘要。若附件为 ZIP 文件，说明当前家庭已开启备份加密，请使用备份配置中的密码提示确认解压密码。\n\n如您未进行相关操作，请及时联系系统管理员。", scene, fileName, sentAt.Format("2006-01-02 15:04:05"))
}

func mapToJSONMap(value map[string]string) models.JSONMap {
	out := models.JSONMap{}
	for key, item := range value {
		out[key] = item
	}
	return out
}
