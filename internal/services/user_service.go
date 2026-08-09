package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/models"
	"backend/internal/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const emailVerificationTokenTTL = 15 * time.Minute

func verificationCodeKey(email string) string {
	return fmt.Sprintf("pato:email-verification:%s", email)
}

type UserService struct {
	DB           *gorm.DB
	Config       *config.Config
	EmailService *EmailService
	Valkey       *redis.Client
}

func NewUserService(db *gorm.DB, cfg *config.Config, emailService *EmailService, valkey *redis.Client) *UserService {
	return &UserService{DB: db, Config: cfg, EmailService: emailService, Valkey: valkey}
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
	code, err := utils.GenerateNumericCode()
	if err != nil {
		log.Printf("failed to generate verification code for %s: %v", user.Email, err)
		return
	}

	ctx := context.Background()
	if err := s.Valkey.Set(ctx, verificationCodeKey(user.Email), code, emailVerificationTokenTTL).Err(); err != nil {
		log.Printf("failed to store verification token for %s: %v", user.Email, err)
		return
	}

	if err := s.EmailService.SendVerificationEmail(user.Username, user.Email, code); err != nil {
		log.Printf("failed to send verification email to %s: %v", user.Email, err)
	}
}

func (s *UserService) VerifyEmail(email, token string) error {
	ctx := context.Background()
	key := verificationCodeKey(email)

	storedCode, err := s.Valkey.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("invalid or expired token")
		}
		return fmt.Errorf("valkey error: %w", err)
	}

	if storedCode != token {
		return fmt.Errorf("invalid or expired token")
	}

	if err := s.DB.Model(&models.User{}).Where("email = ?", email).
		Update("email_verified", true).Error; err != nil {
		return fmt.Errorf("failed to mark email as verified: %w", err)
	}

	if err := s.Valkey.Del(ctx, key).Err(); err != nil {
		log.Printf("failed to delete used verification token for %s: %v", email, err)
	}

	return nil
}

func (s *UserService) Login(req *dto.LoginRequest) (string, *models.User, error) {
	var user models.User

	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, fmt.Errorf("invalid email or password")
		}

		return "", nil, fmt.Errorf("database error: %w", err)
	}

	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		return "", nil, fmt.Errorf("invalid emaill oor password")
	}

	if !user.EmailVerified {
		return "", nil, fmt.Errorf("email not verified")
	}

	token, err := utils.GenerateJWT(&user, s.Config.JWT)

	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, &user, nil

}

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}

	var user models.User

	if err := s.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}

		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}
