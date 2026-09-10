package handlers

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	token, err := h.userService.Login(&req)

	if err != nil {
		if errors.Is(err, services.ErrEmailNotVerified) {
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Error: err.Error(),
				Code:  "EMAIL_NOT_VERIFIED",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "LOGIN_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{
		Token:     token,
		ExpiresIn: int64(config.SessionTTL.Seconds()),
	})
}

func (h *UsersHandler) Logout(c *gin.Context) {

	token := c.GetString("sessionToken")

	if token == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Session token not found",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if err := h.userService.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "LOGOUT_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

func (h *UsersHandler) Me(c *gin.Context) {
	userID, exists := c.Get("userID")

	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "User not authenticated",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userIDString, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Invalid user information",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	user, err := h.userService.GetUserByID(userIDString)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "USER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *UsersHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "USERS_LIST_FAILED",
		})
		return
	}

	responses := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, dto.UserResponse{
			ID:        user.ID.String(),
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, responses)
}

func (h *UsersHandler) UpdateEmail(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Invalid user information in token",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.UpdateEmailRequest
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

	user, err := h.userService.UpdateEmail(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "EMAIL_UPDATE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *UsersHandler) UpdateUsername(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Invalid user information in token",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.UpdateUsernameRequest
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

	user, err := h.userService.UpdateUsername(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "USERNAME_UPDATE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *UsersHandler) ChangePassword(c *gin.Context) {
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Invalid user information in token",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.ChangePasswordRequest
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

	if err := h.userService.ChangePassword(userID, &req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "PASSWORD_CHANGE_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

const passwordResetTokenTTL = 15 * time.Minute

func (h *UsersHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest

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

	if err := h.userService.ForgotPassword(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "PASSWORD_RESET_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If the email is registered, a password reset token has been sent",
	})
}

func (h *UsersHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest

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

	if err := h.userService.ResetPassword(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "PASSWORD_RESET_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}
