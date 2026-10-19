package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultServerPort           = "8080"
	DefaultServerHost           = ""
	DefaultDBHost               = "localhost"
	DefaultDBPort               = "5432"
	DefaultDBUser               = "postgres"
	DefaultDBPassword           = "postgres"
	DefaultDBName               = "miniot"
	DefaultDBSSLMode            = "disable"
	DefaultRedisHost            = "localhost"
	DefaultRedisPort            = "6379"
	DefaultEnvironment          = "development"
	DefaultJWTSecret            = "miniot_super_secret_jwt_key_change_in_production"
	DefaultAccessTokenTTL       = 15 * time.Minute
	DefaultRefreshTokenTTL      = 7 * 24 * time.Hour
	DefaultLoginRateLimitMax    = 5
	DefaultLoginRateLimitWindow = 1 * time.Minute
)

type Config struct {
	Environment            string
	ServerPort             string
	ServerHost             string
	DatabaseHost           string
	DatabasePort           string
	DatabaseUser           string
	DatabasePassword       string
	DatabaseName           string
	DatabaseSSLMode        string
	RedisHost              string
	RedisPort              string
	RedisPassword          string
	CORSAllowedOrigins     []string
	JWTSecret              string
	AccessTokenTTL         time.Duration
	RefreshTokenTTL        time.Duration
	LoginRateLimitAttempts int
	LoginRateLimitWindow   time.Duration
}

// LoadConfig loads environment variables with sane defaults.
func LoadConfig() *Config {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	corsEnv := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://localhost:80")
	allowedOrigins := strings.Split(corsEnv, ",")

	return &Config{
		Environment:            getEnv("APP_ENV", DefaultEnvironment),
		ServerPort:             getEnv("API_PORT", DefaultServerPort),
		ServerHost:             getEnv("API_HOST", DefaultServerHost),
		DatabaseHost:           getEnv("DB_HOST", DefaultDBHost),
		DatabasePort:           getEnv("DB_PORT", DefaultDBPort),
		DatabaseUser:           getEnv("DB_USER", DefaultDBUser),
		DatabasePassword:       getEnv("DB_PASSWORD", DefaultDBPassword),
		DatabaseName:           getEnv("DB_NAME", DefaultDBName),
		DatabaseSSLMode:        getEnv("DB_SSLMODE", DefaultDBSSLMode),
		RedisHost:              getEnv("REDIS_HOST", DefaultRedisHost),
		RedisPort:              getEnv("REDIS_PORT", DefaultRedisPort),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		CORSAllowedOrigins:     allowedOrigins,
		JWTSecret:              getEnv("JWT_SECRET", DefaultJWTSecret),
		AccessTokenTTL:         parseDuration(getEnv("ACCESS_TOKEN_TTL", "15m"), DefaultAccessTokenTTL),
		RefreshTokenTTL:        parseDuration(getEnv("REFRESH_TOKEN_TTL", "168h"), DefaultRefreshTokenTTL),
		LoginRateLimitAttempts: parseInt(getEnv("LOGIN_RATE_LIMIT_ATTEMPTS", "5"), DefaultLoginRateLimitMax),
		LoginRateLimitWindow:   parseDuration(getEnv("LOGIN_RATE_LIMIT_WINDOW", "1m"), DefaultLoginRateLimitWindow),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func parseDuration(val string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}

func parseInt(val string, fallback int) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
