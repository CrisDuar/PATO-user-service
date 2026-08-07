package config

import (
	"fmt"
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServerConfig struct {
	Port        string
	Environment string
	Name        string
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
		Server: ServerConfig{
			Port:        getEnv("APP_PORT", "8080"),
			Environment: getEnv("APP_ENV", "development"),
			Name:        getEnv("APP_NAME", "PATO User Service"),
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

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
