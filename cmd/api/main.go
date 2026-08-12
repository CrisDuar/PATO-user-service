package main

import (
	"context"
	"fmt"
	"log"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handlers"
	"backend/internal/middleware"
	"backend/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	valkeyClient, err := database.ConnectValkey(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to valkey: %v", err)
	}

	emailService := services.NewEmailService(cfg.EmailService.BaseURL)
	userService := services.NewUserService(db, cfg, emailService, valkeyClient)
	usersHandler := handlers.NewUsersHandler(userService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		valkeyStatus := "healthy"
		if err := valkeyClient.Ping(context.Background()).Err(); err != nil {
			valkeyStatus = "unhealthy"
		}

		c.JSON(200, gin.H{
			"status": "healthy",
			"app":    cfg.Server.Name,
			"valkey": valkeyStatus,
		})
	})

	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("/register", usersHandler.Register)
			users.POST("/verify-email", usersHandler.VerifyEmail)
			users.POST("/login", usersHandler.Login)
			users.POST("/forgot-password", usersHandler.ForgotPassword)
			users.POST("/reset-password", usersHandler.ResetPassword)

			protected := users.Group("")
			protected.Use(middleware.AuthMiddleware(userService))
			{
				protected.GET("/me", usersHandler.Me)
				protected.POST("/logout", usersHandler.Logout)
				protected.PATCH("/email", usersHandler.UpdateEmail)
				protected.PATCH("/password", usersHandler.ChangePassword)
			}
		}
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting %s on %s", cfg.Server.Name, addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
