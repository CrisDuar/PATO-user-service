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

type ErrorResponse struct {
	Error       string `json:"error"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

func (r *UserCreateRequest) Validate() error {
	return validate.Struct(r)
}
