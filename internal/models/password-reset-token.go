package models

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`
	TokenHash string    `gorm:"column:token_hash;type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"column:used;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_token"
}
