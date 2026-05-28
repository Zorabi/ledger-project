package database

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ledger-backend/internal/config"
	"ledger-backend/internal/models"
)

func ConnectDatabase() *gorm.DB {
	cfg := config.AppConfig.Database.Postgres
	ensureDatabase(cfg)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect database: %s", err))
	}

	if err := db.AutoMigrate(
		&models.Family{},
		&models.User{},
		&models.UserSession{},
		&models.DictionaryType{},
		&models.DictionaryItem{},
		&models.Contact{},
		&models.GiftRecord{},
		&models.Attachment{},
		&models.BackupConfig{},
		&models.BackupJob{},
		&models.ImportJob{},
		&models.ImportJobError{},
		&models.AuditLog{},
	); err != nil {
		panic(fmt.Sprintf("migrate database: %s", err))
	}
	backfillFamilyAdmins(db)

	slog.Info("database connected and migrated")
	return db
}

func ensureDatabase(cfg config.PostgresConfig) {
	if cfg.DBName == "" || cfg.DBName == "postgres" {
		return
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(cfg.DBName) {
		panic("database name only supports letters, numbers and underscore")
	}
	maintenanceDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=%s TimeZone=Asia/Shanghai",
		cfg.Host, cfg.User, cfg.Password, cfg.Port, cfg.SSLMode)
	db, err := gorm.Open(postgres.Open(maintenanceDSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("connect postgres maintenance database: %s", err))
	}
	var exists int64
	db.Raw("SELECT count(1) FROM pg_database WHERE datname = ?", cfg.DBName).Scan(&exists)
	if exists > 0 {
		return
	}
	quotedName := `"` + strings.ReplaceAll(cfg.DBName, `"`, `""`) + `"`
	if err := db.Exec("CREATE DATABASE " + quotedName).Error; err != nil {
		panic(fmt.Sprintf("create database %s: %s", cfg.DBName, err))
	}
}

func backfillFamilyAdmins(db *gorm.DB) {
	db.Exec(`
		UPDATE families f
		SET admin_user_id = u.id
		FROM users u
		WHERE f.admin_user_id IS NULL
		  AND u.family_id = f.id
		  AND u.role = ?
		  AND u.deleted_at IS NULL
	`, models.RoleFamilyAdmin)
}
