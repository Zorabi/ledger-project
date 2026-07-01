package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        AppMetaConfig
	Server     ServerConfig
	Log        LogConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Security   SecurityConfig
	Audit      AuditConfig
	Bootstrap  BootstrapConfig
	Email      EmailConfig
	Encryption EncryptionConfig
}

type AppMetaConfig struct {
	Name    string
	Version string
}

type ServerConfig struct {
	Port int
	Page PageConfig
}

type PageConfig struct {
	DefaultPageSize int `mapstructure:"default_page_size"`
	DefaultPageNum  int `mapstructure:"default_page_num"`
	MaxPageSize     int `mapstructure:"max_page_size"`
}

type LogConfig struct {
	Level      string
	Console    bool
	File       bool
	FilePath   string `mapstructure:"file_path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
	Compress   bool
}

type DatabaseConfig struct {
	Postgres PostgresConfig
}

type PostgresConfig struct {
	User        string
	Password    string
	Host        string
	Port        int
	DBName      string
	SSLMode     string `mapstructure:"ssl_mode"`
	AutoCreate  bool   `mapstructure:"auto_create"`
	AutoMigrate bool   `mapstructure:"auto_migrate"`
}

type JWTConfig struct {
	Secret           string
	SigningMethod    string `mapstructure:"signing_method"`
	AccessTokenHours int    `mapstructure:"access_token_hours"`
	RefreshTokenDays int    `mapstructure:"refresh_token_days"`
	Issuer           string
}

type SecurityConfig struct {
	LoginMaxAttempts      int `mapstructure:"login_max_attempts"`
	LoginWindowMinutes    int `mapstructure:"login_window_minutes"`
	LoginLockMinutes      int `mapstructure:"login_lock_minutes"`
	LoginCleanupAfterDays int `mapstructure:"login_cleanup_after_days"`
}

type AuditConfig struct {
	RetentionDays int `mapstructure:"retention_days"`
}

type BootstrapConfig struct {
	SuperAdminUsername string `mapstructure:"super_admin_username"`
	SuperAdminPassword string `mapstructure:"super_admin_password"`
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type EncryptionConfig struct {
	AppKey string `mapstructure:"app_key"`
}

var AppConfig Config

func LoadConfig() {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("backend")
	v.AddConfigPath("../backend")

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.page.default_page_size", 10)
	v.SetDefault("server.page.default_page_num", 1)
	v.SetDefault("server.page.max_page_size", 100)
	v.SetDefault("app.name", "礼尚往来")
	v.SetDefault("app.version", "1.6.0.0")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.console", true)
	v.SetDefault("log.file", true)
	v.SetDefault("log.file_path", "logs/ledger-backend.log")
	v.SetDefault("log.max_size_mb", 100)
	v.SetDefault("log.max_backups", 10)
	v.SetDefault("log.max_age_days", 30)
	v.SetDefault("log.compress", true)
	v.SetDefault("database.postgres.auto_create", true)
	v.SetDefault("database.postgres.auto_migrate", true)
	v.SetDefault("jwt.signing_method", "HS256")
	v.SetDefault("jwt.access_token_hours", 2)
	v.SetDefault("jwt.refresh_token_days", 30)
	v.SetDefault("jwt.issuer", "ledger")
	v.SetDefault("security.login_max_attempts", 5)
	v.SetDefault("security.login_window_minutes", 15)
	v.SetDefault("security.login_lock_minutes", 30)
	v.SetDefault("security.login_cleanup_after_days", 7)
	v.SetDefault("audit.retention_days", 30)
	v.SetDefault("bootstrap.super_admin_username", "admin")
	v.SetDefault("bootstrap.super_admin_password", "ChangeMe123!")

	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read config: %w", err))
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	bindEnvs(v, []string{
		"database.postgres.user",
		"database.postgres.password",
		"database.postgres.host",
		"database.postgres.port",
		"database.postgres.dbname",
		"database.postgres.ssl_mode",
		"database.postgres.auto_create",
		"database.postgres.auto_migrate",
		"app.name",
		"app.version",
		"log.level",
		"log.console",
		"log.file",
		"log.file_path",
		"log.max_size_mb",
		"log.max_backups",
		"log.max_age_days",
		"log.compress",
		"jwt.secret",
		"jwt.access_token_hours",
		"jwt.refresh_token_days",
		"security.login_max_attempts",
		"security.login_window_minutes",
		"security.login_lock_minutes",
		"security.login_cleanup_after_days",
		"audit.retention_days",
		"bootstrap.super_admin_username",
		"bootstrap.super_admin_password",
		"email.host",
		"email.port",
		"email.username",
		"email.password",
		"email.from",
		"encryption.app_key",
	})
	v.BindEnv("encryption.app_key", "APP_ENCRYPTION_KEY")

	if err := v.Unmarshal(&AppConfig); err != nil {
		panic(fmt.Errorf("unmarshal config: %w", err))
	}
	if AppConfig.Server.Page.MaxPageSize == 0 {
		AppConfig.Server.Page.MaxPageSize = 100
	}
	if AppConfig.Log.FilePath == "" {
		AppConfig.Log.FilePath = "logs/ledger-backend.log"
	}
	if !filepath.IsAbs(AppConfig.Log.FilePath) {
		AppConfig.Log.FilePath = filepath.Clean(filepath.Join(filepath.Dir(v.ConfigFileUsed()), AppConfig.Log.FilePath))
	}
	if AppConfig.Log.MaxSizeMB <= 0 {
		AppConfig.Log.MaxSizeMB = 100
	}
	if AppConfig.Log.MaxBackups <= 0 {
		AppConfig.Log.MaxBackups = 10
	}
	if AppConfig.Log.MaxAgeDays <= 0 {
		AppConfig.Log.MaxAgeDays = 30
	}
	if AppConfig.JWT.Secret == "" {
		panic("jwt.secret is required")
	}
	if AppConfig.JWT.AccessTokenHours <= 0 {
		AppConfig.JWT.AccessTokenHours = 2
	}
	if AppConfig.JWT.RefreshTokenDays <= 0 {
		AppConfig.JWT.RefreshTokenDays = 30
	}
	if AppConfig.Security.LoginMaxAttempts <= 0 {
		AppConfig.Security.LoginMaxAttempts = 5
	}
	if AppConfig.Security.LoginWindowMinutes <= 0 {
		AppConfig.Security.LoginWindowMinutes = 15
	}
	if AppConfig.Security.LoginLockMinutes <= 0 {
		AppConfig.Security.LoginLockMinutes = 30
	}
	if AppConfig.Security.LoginCleanupAfterDays <= 0 {
		AppConfig.Security.LoginCleanupAfterDays = 7
	}
	AppConfig.Audit.RetentionDays = normalizeAuditRetentionDays(AppConfig.Audit.RetentionDays)
}

func normalizeAuditRetentionDays(days int) int {
	switch days {
	case 30, 60, 90:
		return days
	default:
		return 30
	}
}

func bindEnvs(v *viper.Viper, keys []string) {
	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}
