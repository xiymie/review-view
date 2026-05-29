package model

import "time"

type UserRole string

const (
	UserRoleSuperAdmin UserRole = "super_admin"
	UserRoleAdmin      UserRole = "admin"
	UserRoleNormal     UserRole = "user"
)

type User struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username            string    `gorm:"uniqueIndex;not null"     json:"username"`
	PasswordHash        string    `gorm:"not null"                 json:"-"`
	Role                UserRole  `gorm:"not null;default:'user'"  json:"role"`
	Email               string    `json:"email"`
	Phone               string    `json:"phone"`
	Position            string    `json:"position"`
	Remark              string    `json:"remark"`
	NotifyEnabled       bool      `gorm:"not null;default:false"   json:"notify_enabled"`
	NotifyEmails        string    `json:"notify_emails"`
	NotifyWecomWebhook  string    `json:"notify_wecom_webhook"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
