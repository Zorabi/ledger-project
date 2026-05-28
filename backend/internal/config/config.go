package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Bootstrap  BootstrapConfig
	SMTP       SMTPConfig
	Encryption EncryptionConfig
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

type DatabaseConfig struct {
	Postgres PostgresConfig
}

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
	SSLMode  string `mapstructure:"ssl_mode"`
}

type JWTConfig struct {
	Secret           string
	SigningMethod    string        `mapstructure:"signing_method"`
	AccessTokenHours time.Duration `mapstructure:"access_token_hours"`
	RefreshTokenDays time.Duration `mapstructure:"refresh_token_days"`
	Issuer           string
}

type BootstrapConfig struct {
	SuperAdminUsername string `mapstructure:"super_admin_username"`
	SuperAdminPassword string `mapstructure:"super_admin_password"`
}

type SMTPConfig struct {
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
	v.SetDefault("jwt.signing_method", "HS256")
	v.SetDefault("jwt.access_token_hours", 2)
	v.SetDefault("jwt.refresh_token_days", 30)
	v.SetDefault("jwt.issuer", "ledger")
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
		"jwt.secret",
		"jwt.access_token_hours",
		"jwt.refresh_token_days",
		"bootstrap.super_admin_username",
		"bootstrap.super_admin_password",
		"smtp.host",
		"smtp.port",
		"smtp.username",
		"smtp.password",
		"smtp.from",
		"encryption.app_key",
	})
	v.BindEnv("encryption.app_key", "APP_ENCRYPTION_KEY")

	if err := v.Unmarshal(&AppConfig); err != nil {
		panic(fmt.Errorf("unmarshal config: %w", err))
	}
	if AppConfig.Server.Page.MaxPageSize == 0 {
		AppConfig.Server.Page.MaxPageSize = 100
	}
	if AppConfig.JWT.Secret == "" {
		panic("jwt.secret is required")
	}
}

func bindEnvs(v *viper.Viper, keys []string) {
	for _, key := range keys {
		_ = v.BindEnv(key)
	}
}
