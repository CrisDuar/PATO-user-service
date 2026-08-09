package utils

import (
	"backend/internal/config"
	"backend/internal/models"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(user *models.User, cfg config.JWTConfig) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"subs":     user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"iat":      now.Unix(),
		"exp":      now.Add(cfg.Expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

func ValidateJWT(tokenString string, cfg config.JWTConfig) (jwt.MapClaims, error) {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %w",
					token.Method.Alg(),
				)
			}
			return []byte(cfg.Secret), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid JWT: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims")
	}
	return claims, nil

}
