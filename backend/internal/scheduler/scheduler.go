package scheduler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"ledger-backend/internal/config"
	"ledger-backend/internal/exporter"
	"ledger-backend/internal/mailer"
	"ledger-backend/internal/models"
	"ledger-backend/internal/utils"
)

type Scheduler struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Scheduler {
	return &Scheduler{db: db}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.loop(ctx, 24*time.Hour)
}

func (s *Scheduler) loop(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := s.CleanupSessions(); err != nil {
				slog.Error("cleanup sessions failed", "err", err)
			}
			if err := s.RunDueBackups(); err != nil {
				slog.Error("run due backups failed", "err", err)
			}
			timer.Reset(interval)
		}
	}
}

func (s *Scheduler) CleanupSessions() error {
	now := time.Now()
	sessionCutoff := now.AddDate(0, 0, -7)
	if err := s.db.Exec(
		"DELETE FROM user_sessions WHERE expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)",
		now, sessionCutoff,
	).Error; err != nil {
		return err
	}
	throttleCutoff := now.AddDate(0, 0, -config.AppConfig.Security.LoginCleanupAfterDays)
	return s.db.Where("updated_at < ?", throttleCutoff).Delete(&models.AuthThrottle{}).Error
}

func (s *Scheduler) RunDueBackups() error {
	var configs []models.BackupConfig
	if err := s.db.Where("enabled = true").Find(&configs).Error; err != nil {
		return err
	}
	for _, cfg := range configs {
		if !backupDue(cfg, time.Now()) {
			continue
		}
		if err := s.runBackup(cfg); err != nil {
			slog.Error("scheduled backup failed", "family_id", cfg.FamilyID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) runBackup(cfg models.BackupConfig) error {
	started := time.Now()
	job := models.BackupJob{FamilyID: cfg.FamilyID, TriggerType: "scheduled", Status: "running", RecipientEmails: cfg.RecipientEmails, StartedAt: started}
	if err := s.db.Create(&job).Error; err != nil {
		return err
	}
	backup, err := exporter.GenerateFamilyBackup(s.db, cfg.FamilyID)
	finished := time.Now()
	if err != nil {
		s.db.Model(&job).Updates(map[string]any{"status": "failed", "finished_at": finished, "error_message": err.Error()})
		s.db.Model(&cfg).Updates(map[string]any{"last_sent_at": finished, "last_status": "failed"})
		return err
	}
	content, fileName, contentType, err := scheduledAttachment(cfg, backup.FileName, backup.Content)
	if err != nil {
		s.db.Model(&job).Updates(map[string]any{"status": "failed", "finished_at": finished, "error_message": err.Error()})
		s.db.Model(&cfg).Updates(map[string]any{"last_sent_at": finished, "last_status": "failed"})
		return err
	}
	sum := sha256.Sum256(content)
	if err := mailer.SendBackup(cfg.RecipientEmails, "礼尚往来定时备份", backupMailBody(cfg), mailer.Attachment{FileName: fileName, ContentType: contentType, Content: content}); err != nil {
		s.db.Model(&job).Updates(map[string]any{"status": "failed", "file_name": fileName, "file_size": len(content), "file_sha256": fmt.Sprintf("%x", sum[:]), "finished_at": finished, "error_message": err.Error()})
		s.db.Model(&cfg).Updates(map[string]any{"last_sent_at": finished, "last_status": "failed"})
		return err
	}
	s.db.Model(&job).Updates(map[string]any{"status": "success", "file_name": fileName, "file_size": len(content), "file_sha256": fmt.Sprintf("%x", sum[:]), "finished_at": finished})
	s.db.Model(&cfg).Updates(map[string]any{"last_sent_at": finished, "last_status": "success"})
	return nil
}

func backupDue(cfg models.BackupConfig, now time.Time) bool {
	if cfg.LastSentAt == nil {
		return true
	}
	switch cfg.Frequency {
	case "daily":
		return now.Sub(*cfg.LastSentAt) >= 24*time.Hour
	case "weekly":
		return now.Sub(*cfg.LastSentAt) >= 7*24*time.Hour
	case "monthly":
		return cfg.LastSentAt.AddDate(0, 1, 0).Before(now)
	default:
		return now.Sub(*cfg.LastSentAt) >= 7*24*time.Hour
	}
}

func backupMailBody(cfg models.BackupConfig) string {
	body := "附件为礼尚往来家庭完整备份。"
	if cfg.EncryptEnabled && cfg.ZipPasswordHint != "" {
		body += "ZIP 密码提示：" + cfg.ZipPasswordHint
	}
	return body
}

func scheduledAttachment(cfg models.BackupConfig, fileName string, content []byte) ([]byte, string, string, error) {
	if !cfg.EncryptEnabled {
		return content, fileName, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	}
	password, err := utils.DecryptText(cfg.ZipPasswordEncrypted, config.AppConfig.Encryption.AppKey)
	if err != nil || password == "" {
		return nil, "", "", fmt.Errorf("备份 ZIP 加密已启用，但密码不可用")
	}
	zipContent, zipName, err := exporter.EncryptZip(fileName, content, password)
	if err != nil {
		return nil, "", "", err
	}
	return zipContent, zipName, "application/zip", nil
}
