package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application runtime settings.
type Config struct {
	ServerPort string
	MySQLDSN   string
	RedisAddr  string
	RedisPass  string
	RedisDB    int
	JWTSecret  string
}

// Load returns configuration values from environment with defaults.
func Load() Config {
	// Load .env into process environment before reading variables.
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	return Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		MySQLDSN: getEnv("MYSQL_DSN", "root:password@tcp(127.0.0.1:3306)/schedule_system?charset=utf8mb4&parseTime=True&loc=Local"),
		RedisAddr:  getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:    redisDB,
		JWTSecret:  getEnv("JWT_SECRET", "change-me"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
