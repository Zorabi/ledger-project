package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	RoleSuperAdmin  = "super_admin"
	RoleFamilyAdmin = "family_admin"
	RoleMember      = "member"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"

	DirectionIn  = "in"
	DirectionOut = "out"

	ReciprocityNone    = "none"
	ReciprocityPending = "pending"
	ReciprocityDone    = "done"
)

type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported StringArray scan type %T", value)
	}
	return json.Unmarshal(raw, s)
}

type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("unsupported JSONMap scan type %T", value)
	}
	return json.Unmarshal(raw, m)
}

type Family struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"size:120;not null"`
	ContactName  string         `json:"contact_name" gorm:"size:80"`
	ContactEmail string         `json:"contact_email" gorm:"size:160"`
	Remark       string         `json:"remark" gorm:"type:text"`
	Status       string         `json:"status" gorm:"size:20;not null;default:enabled;index"`
	AdminUserID  *uint          `json:"admin_user_id" gorm:"index"`
	AdminUser    *User          `json:"admin_user,omitempty" gorm:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type User struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	FamilyID           *uint          `json:"family_id" gorm:"index"`
	Family             *Family        `json:"family,omitempty"`
	Username           string         `json:"username" gorm:"size:80;uniqueIndex;not null"`
	DisplayName        string         `json:"display_name" gorm:"size:80;not null"`
	Email              string         `json:"email" gorm:"size:160"`
	PasswordHash       string         `json:"-" gorm:"size:255;not null"`
	Role               string         `json:"role" gorm:"size:30;not null;index"`
	MustChangePassword bool           `json:"must_change_password" gorm:"not null;default:true"`
	Status             string         `json:"status" gorm:"size:20;not null;default:enabled;index"`
	LastLoginAt        *time.Time     `json:"last_login_at"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserSession struct {
	ID               uint       `json:"id" gorm:"primaryKey"`
	UserID           uint       `json:"user_id" gorm:"not null;index"`
	RefreshTokenHash string     `json:"-" gorm:"size:128;not null;uniqueIndex"`
	ExpiresAt        time.Time  `json:"expires_at" gorm:"not null;index"`
	RevokedAt        *time.Time `json:"revoked_at" gorm:"index"`
	IP               string     `json:"ip" gorm:"size:80"`
	UserAgent        string     `json:"user_agent" gorm:"size:255"`
	CreatedAt        time.Time  `json:"created_at"`
}

type AuthThrottle struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Scope           string     `json:"scope" gorm:"size:30;not null;uniqueIndex:idx_auth_throttle_scope_key"`
	Key             string     `json:"key" gorm:"size:160;not null;uniqueIndex:idx_auth_throttle_scope_key"`
	FailedCount     int        `json:"failed_count" gorm:"not null;default:0"`
	WindowStartedAt time.Time  `json:"window_started_at" gorm:"not null;index"`
	LastFailedAt    time.Time  `json:"last_failed_at" gorm:"not null;index"`
	LockedUntil     *time.Time `json:"locked_until" gorm:"index"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type DictionaryType struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Code              string    `json:"code" gorm:"size:60;uniqueIndex;not null"`
	Name              string    `json:"name" gorm:"size:80;not null"`
	Description       string    `json:"description" gorm:"size:255"`
	AllowFamilyCustom bool      `json:"allow_family_custom" gorm:"not null;default:true"`
	SortOrder         int       `json:"sort_order" gorm:"not null;default:0"`
	Enabled           bool      `json:"enabled" gorm:"not null;default:true;index"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DictionaryItem struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TypeID    uint           `json:"type_id" gorm:"not null;index"`
	Type      DictionaryType `json:"type,omitempty"`
	FamilyID  *uint          `json:"family_id" gorm:"index"`
	Label     string         `json:"label" gorm:"size:80;not null"`
	Value     string         `json:"value" gorm:"size:80;not null"`
	SortOrder int            `json:"sort_order" gorm:"not null;default:0"`
	Enabled   bool           `json:"enabled" gorm:"not null;default:true;index"`
	IsSystem  bool           `json:"is_system" gorm:"not null;default:false;index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type Contact struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	FamilyID     uint           `json:"family_id" gorm:"not null;index"`
	Name         string         `json:"name" gorm:"size:120;not null;index"`
	ContactType  string         `json:"contact_type" gorm:"size:80;not null;index"`
	RelationType string         `json:"relation_type" gorm:"size:80;not null;index"`
	Phone        string         `json:"phone" gorm:"size:40"`
	Address      string         `json:"address" gorm:"size:255"`
	Remark       string         `json:"remark" gorm:"type:text"`
	Status       string         `json:"status" gorm:"size:20;not null;default:enabled;index"`
	CreatedBy    uint           `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type GiftRecord struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	FamilyID          uint           `json:"family_id" gorm:"not null;index"`
	ContactID         uint           `json:"contact_id" gorm:"not null;index"`
	Contact           Contact        `json:"contact,omitempty"`
	OccasionType      string         `json:"occasion_type" gorm:"size:80;not null;index"`
	Direction         string         `json:"direction" gorm:"size:10;not null;index"`
	AmountCents       int64          `json:"amount_cents" gorm:"not null;default:0"`
	GiftDescription   string         `json:"gift_description" gorm:"type:text"`
	OccurredOn        time.Time      `json:"occurred_on" gorm:"type:date;not null;index"`
	HandledBy         string         `json:"handled_by" gorm:"size:80"`
	ReciprocityStatus string         `json:"reciprocity_status" gorm:"size:20;not null;default:none;index"`
	RelatedRecordID   *uint          `json:"related_record_id" gorm:"index"`
	Remark            string         `json:"remark" gorm:"type:text"`
	CreatedBy         uint           `json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

type Attachment struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	FamilyID   uint           `json:"family_id" gorm:"not null;index"`
	RecordID   uint           `json:"record_id" gorm:"not null;index"`
	FileName   string         `json:"file_name" gorm:"size:255;not null"`
	StorageKey string         `json:"storage_key" gorm:"size:255;not null"`
	FileType   string         `json:"file_type" gorm:"size:80"`
	FileSize   int64          `json:"file_size"`
	UploadedBy uint           `json:"uploaded_by"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type BackupConfig struct {
	ID                   uint        `json:"id" gorm:"primaryKey"`
	FamilyID             uint        `json:"family_id" gorm:"uniqueIndex;not null"`
	Enabled              bool        `json:"enabled" gorm:"not null;default:false"`
	Frequency            string      `json:"frequency" gorm:"size:20;not null;default:weekly"`
	RecipientEmails      StringArray `json:"recipient_emails" gorm:"type:jsonb;not null;default:'[]'"`
	EncryptEnabled       bool        `json:"encrypt_enabled" gorm:"not null;default:false"`
	ZipPasswordEncrypted string      `json:"-" gorm:"type:text"`
	ZipPasswordHint      string      `json:"zip_password_hint" gorm:"size:255"`
	LastSentAt           *time.Time  `json:"last_sent_at"`
	LastStatus           string      `json:"last_status" gorm:"size:30"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

type BackupJob struct {
	ID              uint        `json:"id" gorm:"primaryKey"`
	FamilyID        uint        `json:"family_id" gorm:"not null;index"`
	TriggerType     string      `json:"trigger_type" gorm:"size:30;not null"`
	Status          string      `json:"status" gorm:"size:30;not null;index"`
	FileName        string      `json:"file_name" gorm:"size:255"`
	FileSize        int64       `json:"file_size"`
	FileSHA256      string      `json:"file_sha256" gorm:"size:64"`
	RecipientEmails StringArray `json:"recipient_emails" gorm:"type:jsonb;not null;default:'[]'"`
	StartedAt       time.Time   `json:"started_at"`
	FinishedAt      *time.Time  `json:"finished_at"`
	ErrorMessage    string      `json:"error_message" gorm:"type:text"`
	CreatedBy       *uint       `json:"created_by"`
}

type ImportJob struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	FamilyID       uint       `json:"family_id" gorm:"not null;index"`
	OperatorID     uint       `json:"operator_id" gorm:"not null;index"`
	Status         string     `json:"status" gorm:"size:30;not null;index"`
	DryRun         bool       `json:"dry_run" gorm:"not null;default:true"`
	SourceFileName string     `json:"source_file_name" gorm:"size:255"`
	SourcePayload  JSONMap    `json:"-" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedCount   int        `json:"created_count"`
	UpdatedCount   int        `json:"updated_count"`
	SkippedCount   int        `json:"skipped_count"`
	ErrorCount     int        `json:"error_count"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	ErrorMessage   string     `json:"error_message" gorm:"type:text"`
}

type ImportJobError struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	ImportJobID  uint    `json:"import_job_id" gorm:"not null;index"`
	SheetName    string  `json:"sheet_name" gorm:"size:80"`
	RowNumber    int     `json:"row_number"`
	FieldName    string  `json:"field_name" gorm:"size:80"`
	ErrorMessage string  `json:"error_message" gorm:"type:text"`
	RawData      JSONMap `json:"raw_data" gorm:"type:jsonb;not null;default:'{}'"`
}

type AuditLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	FamilyID     *uint     `json:"family_id" gorm:"index"`
	OperatorID   *uint     `json:"operator_id" gorm:"index"`
	Operator     *User     `json:"operator,omitempty" gorm:"foreignKey:OperatorID"`
	Action       string    `json:"action" gorm:"size:80;not null;index"`
	ResourceType string    `json:"resource_type" gorm:"size:80;not null;index"`
	ResourceID   string    `json:"resource_id" gorm:"size:80"`
	Success      bool      `json:"success" gorm:"not null;default:true;index"`
	IP           string    `json:"ip" gorm:"size:80"`
	UserAgent    string    `json:"user_agent" gorm:"size:255"`
	Detail       JSONMap   `json:"detail" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time `json:"created_at"`
}

func NormalizeRole(role string) string {
	return strings.TrimSpace(strings.ToLower(role))
}
