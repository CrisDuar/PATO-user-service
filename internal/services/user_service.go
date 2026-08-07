package services

import (
	"fmt"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/models"
	"backend/internal/utils"

	"gorm.io/gorm"
)

type UserService struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewUserService(db *gorm.DB, cfg *config.Config) *UserService {
	return &UserService{DB: db, Config: cfg}
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

	return user, nil
}
