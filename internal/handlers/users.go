package handlers

import (
	"net/http"

	"backend/internal/dto"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

type UsersHandler struct {
	userService *services.UserService
}

func NewUsersHandler(us *services.UserService) *UsersHandler {
	return &UsersHandler{userService: us}
}

func (h *UsersHandler) Register(c *gin.Context) {
	var req dto.UserCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Invalid request format",
			Code:        "INVALID_REQUEST",
			Description: err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Validation failed",
			Code:        "VALIDATION_ERROR",
			Description: err.Error(),
		})
		return
	}

	user, err := h.userService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "USER_CREATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, dto.UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *UsersHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Invalid request format",
			Code:        "INVALID_REQUEST",
			Description: err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Validation failed",
			Code:        "VALIDATION_ERROR",
			Description: err.Error(),
		})
		return
	}

	if err := h.userService.VerifyEmail(req.Email, req.Token); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "EMAIL_VERIFICATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *UsersHandler) Login(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Invalid request format",
			Code:        "INVALID_REQUEST",
			Description: err.Error(),
		})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:       "Validation failed",
			Code:        "VALIDATION_ERROR",
			Description: err.Error(),
		})
		return
	}
	token, user, err := h.userService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "LOGIN_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{
		Token: token,
		User: dto.UserResponse{
			ID:        user.ID.String(),
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05z"),
		},
	})
}
