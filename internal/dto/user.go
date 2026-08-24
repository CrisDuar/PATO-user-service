package dto

import "github.com/go-playground/validator/v10"

var validate = validator.New()

type UserCreateRequest struct {
	Username        string `json:"username" validate:"required,min=2,max=50"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

type UpdateUserRequest struct {
	Username string `json:"username" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
}

type UpdateEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" validate:"required"`
	NewPassword        string `json:"new_password" validate:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

func (r *ForgotPasswordRequest) Validate() error {
	return validate.Struct(r)
}

func (r *ResetPasswordRequest) Validate() error {
	return validate.Struct(r)
}

func (r *UserCreateRequest) Validate() error {
	return validate.Struct(r)
}

func (r *LoginRequest) Validate() error {
	return validate.Struct(r)
}

func (r *VerifyEmailRequest) Validate() error {
	return validate.Struct(r)
}

func (r *UpdateUserRequest) Validate() error {
	return validate.Struct(r)
}

func (r *ChangePasswordRequest) Validate() error {
	return validate.Struct(r)
}

func (r *UpdateEmailRequest) Validate() error {
	return validate.Struct(r)
}
