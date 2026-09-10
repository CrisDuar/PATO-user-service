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

var ErrEmailNotVerified = errors.New("email not verified")

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

func (s *UserService) Login(
	req *dto.LoginRequest,
) (string, error) {

	var user models.User

	if err := s.DB.
		Where("email = ?", req.Email).
		First(&user).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("invalid email or password")
		}

		return "", fmt.Errorf(
			"database error: %w",
			err,
		)
	}

	if !utils.VerifyPassword(
		user.PasswordHash,
		req.Password,
	) {
		return "", fmt.Errorf("invalid email or password")
	}

	if !user.EmailVerified {
		return "", ErrEmailNotVerified
	}

	token, err := utils.GenerateSessionToken()

	if err != nil {
		return "", err
	}

	tokenHash := utils.HashToken(token)

	key := fmt.Sprintf(
		"session:%s",
		tokenHash,
	)

	err = s.Valkey.Set(
		context.Background(),
		key,
		user.ID.String(),
		config.SessionTTL,
	).Err()

	if err != nil {
		return "", fmt.Errorf(
			"failed to create session: %w",
			err,
		)
	}

	return token, nil
}

func (s *UserService) ListUsers() ([]models.User, error) {
	var users []models.User

	if err := s.DB.Order("created_at desc").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return users, nil
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

func (s *UserService) UpdateEmail(userID uuid.UUID, req *dto.UpdateEmailRequest) (*models.User, error) {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid password")
	}

	if req.NewEmail == user.Email {
		return nil, fmt.Errorf("new email must be different from current email")
	}

	var existingUser models.User
	if err := s.DB.Where("email = ?", req.NewEmail).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("database error: %w", err)
	}

	user.Email = req.NewEmail
	user.EmailVerified = false

	if err := s.DB.Model(&user).Select("Email", "EmailVerified").Updates(user).Error; err != nil {
		return nil, fmt.Errorf("failed to update email: %w", err)
	}

	s.issueVerificationEmail(&user)

	return &user, nil
}

func (s *UserService) UpdateUsername(userID uuid.UUID, req *dto.UpdateUsernameRequest) (*models.User, error) {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if req.NewUsername == user.Username {
		return nil, fmt.Errorf("new username must be different from current username")
	}

	user.Username = req.NewUsername

	if err := s.DB.Model(&user).Select("Username").Updates(user).Error; err != nil {
		return nil, fmt.Errorf("failed to update username: %w", err)
	}

	return &user, nil
}

func (s *UserService) ChangePassword(userID uuid.UUID, req *dto.ChangePasswordRequest) error {
	var user models.User
	if err := s.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("database error: %w", err)
	}

	if !utils.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
		return fmt.Errorf("invalid current password")
	}

	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	if utils.VerifyPassword(user.PasswordHash, req.NewPassword) {
		return fmt.Errorf("new password must be different from current password")
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.DB.Model(&user).Update("password_hash", hashedPassword).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

const passwordResetTokenTTL = 15 * time.Minute

func (s *UserService) ForgotPassword(email string) error {
	var user models.User

	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No revelamos si el correo está registrado.
			return nil
		}

		return fmt.Errorf("database error: %w", err)
	}

	token, err := utils.GenerateNumericCode()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	tokenHash := utils.HashToken(token)

	key := fmt.Sprintf("password_reset:%s", tokenHash)

	err = s.Valkey.Set(
		context.Background(),
		key,
		user.ID.String(),
		passwordResetTokenTTL,
	).Err()

	if err != nil {
		return fmt.Errorf("failed to store reset token: %w", err)
	}

	if err := s.EmailService.SendPasswordResetEmail(
		user.Username,
		user.Email,
		token,
	); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	return nil
}

func (s *UserService) ResetPassword(req *dto.ResetPasswordRequest) error {
	if err := utils.ValidatePassword(req.Password); err != nil {
		return err
	}

	tokenHash := utils.HashToken(req.Token)

	key := fmt.Sprintf("password_reset:%s", tokenHash)

	userID, err := s.Valkey.Get(
		context.Background(),
		key,
	).Result()

	if err == redis.Nil {
		return fmt.Errorf("invalid or expired token")
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve reset token: %w", err)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return err
	}

	if err := s.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", hashedPassword).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// El token solo puede utilizarse una vez.
	if err := s.Valkey.Del(
		context.Background(),
		key,
	).Err(); err != nil {
		return fmt.Errorf("failed to invalidate reset token: %w", err)
	}

	return nil
}
func (s *UserService) ValidateSession(
	token string,
) (string, error) {

	tokenHash := utils.HashToken(token)

	key := fmt.Sprintf(
		"session:%s",
		tokenHash,
	)

	ctx := context.Background()

	userID, err := s.Valkey.Get(ctx, key).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf(
				"session expired or invalid",
			)
		}

		return "", fmt.Errorf(
			"failed to validate session: %w",
			err,
		)
	}

	// Renovamos el TTL porque el usuario acaba
	// de tener actividad.
	if err := s.Valkey.Expire(
		ctx,
		key,
		config.SessionTTL,
	).Err(); err != nil {
		return "", fmt.Errorf(
			"failed to refresh session: %w",
			err,
		)
	}

	return userID, nil
}

func (s *UserService) Logout(token string) error {

	tokenHash := utils.HashToken(token)

	key := fmt.Sprintf(
		"session:%s",
		tokenHash,
	)

	if err := s.Valkey.Del(
		context.Background(),
		key,
	).Err(); err != nil {
		return fmt.Errorf(
			"failed to delete session: %w",
			err,
		)
	}

	return nil
}
