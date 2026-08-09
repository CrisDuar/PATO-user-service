package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database     DatabaseConfig
	Valkey       ValkeyConfig
	Server       ServerConfig
	EmailService EmailServiceConfig
	JWT          JWTConfig
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ValkeyConfig struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type ServerConfig struct {
	Port        string
	Environment string
	Name        string
}

type EmailServiceConfig struct {
	BaseURL    string
	AppBaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("APP_DB_IP", "localhost"),
			Port:     getEnv("APP_DB_PORT", "5432"),
			User:     getEnv("APP_DB_USER", "postgres"),
			Password: getEnv("APP_DB_PASSWORD", "postgres"),
			DBName:   getEnv("APP_DB_NAME", "pato_db"),
			SSLMode:  getEnv("APP_DB_SSL_MODE", "disable"),
		},
		Valkey: ValkeyConfig{
			Addr:     getEnv("APP_VALKEY_ADDR", "localhost:6379"),
			Username: getEnv("APP_VALKEY_USER", ""),
			Password: getEnv("APP_VALKEY_PASSWORD", ""),
			DB:       getEnvInt("APP_VALKEY_DB", 0),
		},
		Server: ServerConfig{
			Port:        getEnv("APP_PORT", "8080"),
			Environment: getEnv("APP_ENV", "development"),
			Name:        getEnv("APP_NAME", "PATO User Service"),
		},
		EmailService: EmailServiceConfig{
			BaseURL:    getEnv("EMAIL_SERVICE_URL", "http://localhost:8000"),
			AppBaseURL: getEnv("APP_BASE_URL", "http://localhost:3000"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", ""),
			Expiration: 30 * 24 * time.Hour,
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
