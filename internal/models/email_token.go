package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailVerificationToken struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index" json:"user_id"`
	TokenHash string    `gorm:"column:token_hash;type:varchar(255);not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null" json:"expires_at"`
	Used      bool      `gorm:"column:used;not null;default:false" json:"used"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (EmailVerificationToken) TableName() string {
	return "email_verification_token"
}
