package middleware

import (
	"backend/internal/config"
	"backend/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		claims, err := utils.ValidateJWT(tokenString, cfg.JWT)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		userID, ok := claims["subs"].(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user informmation in token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
