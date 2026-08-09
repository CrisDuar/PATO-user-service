package main

import (
	"context"
	"fmt"
	"log"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handlers"
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

	redisClient, err := database.ConnectRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}

	emailService := services.NewEmailService(cfg.EmailService.BaseURL)
	userService := services.NewUserService(db, cfg, emailService)
	usersHandler := handlers.NewUsersHandler(userService)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		redisStatus := "healthy"
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			redisStatus = "unhealthy"
		}

		c.JSON(200, gin.H{
			"status": "healthy",
			"app":    cfg.Server.Name,
			"redis":  redisStatus,
		})
	})

	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("/register", usersHandler.Register)
			users.POST("/verify-email", usersHandler.VerifyEmail)
			users.POST("/login", usersHandler.Login)
		}
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Starting %s on %s", cfg.Server.Name, addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
