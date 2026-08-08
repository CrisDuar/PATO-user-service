package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/models"
	"backend/internal/utils"

	"gorm.io/gorm"
)

const emailVerificationTokenTTL = 15 * time.Minute

type UserService struct {
	DB           *gorm.DB
	Config       *config.Config
	EmailService *EmailService
}

func NewUserService(db *gorm.DB, cfg *config.Config, emailService *EmailService) *UserService {
	return &UserService{DB: db, Config: cfg, EmailService: emailService}
}

func (s *UserService) CreateUser(req *dto.UserCreateRequest) (*models.User, error) {
	if err := utils.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	var existingUser models.User
	if err := s.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("email already registered")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("database error: %w", err)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
	}

	if err := s.DB.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.issueVerificationEmail(user)

	return user, nil
}

func (s *UserService) issueVerificationEmail(user *models.User) {
	token, err := utils.GenerateVerificationToken()
	if err != nil {
		log.Printf("failed to generate verification token for %s: %v", user.Email, err)
		return
	}

	verificationToken := &models.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: utils.HashToken(token),
		ExpiresAt: time.Now().Add(emailVerificationTokenTTL),
	}

	if err := s.DB.Create(verificationToken).Error; err != nil {
		log.Printf("failed to store verification token for %s: %v", user.Email, err)
		return
	}

	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", s.Config.EmailService.AppBaseURL, token)
	if err := s.EmailService.SendVerificationEmail(user.Username, user.Email, verifyLink); err != nil {
		log.Printf("failed to send verification email to %s: %v", user.Email, err)
	}
}

func (s *UserService) VerifyEmail(token string) error {
	tokenHash := utils.HashToken(token)

	var verificationToken models.EmailVerificationToken
	if err := s.DB.Where("token_hash = ? AND used = ?", tokenHash, false).First(&verificationToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("invalid or already used token")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if time.Now().After(verificationToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", verificationToken.UserID).
			Update("email_verified", true).Error; err != nil {
			return fmt.Errorf("failed to mark email as verified: %w", err)
		}

		if err := tx.Model(&verificationToken).Update("used", true).Error; err != nil {
			return fmt.Errorf("failed to invalidate token: %w", err)
		}

		return nil
	})
}
