package middleware

import (
	"backend/internal/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(
	userService *services.UserService,
) gin.HandlerFunc {

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

		parts := strings.SplitN(
			authHeader,
			" ",
			2,
		)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") {

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "UNAUTHORIZED",
			})

			c.Abort()
			return
		}

		token := parts[1]

		userID, err := userService.ValidateSession(token)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired session",
				"code":  "UNAUTHORIZED",
			})

			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("sessionToken", token)

		c.Next()
	}
}
