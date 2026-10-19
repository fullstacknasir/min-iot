package config

import (
	"os"

	"github.com/joho/godotenv"
)

const (
	DefaultEMQXHost    = "localhost"
	DefaultEMQXPort    = "1883"
	DefaultClientID    = "miniot-ingestor"
	DefaultMQTTTopic   = "devices/+/telemetry"
	DefaultDBHost      = "localhost"
	DefaultDBPort      = "5432"
	DefaultDBUser      = "postgres"
	DefaultDBPassword  = "postgres"
	DefaultDBName      = "miniot"
	DefaultDBSSLMode   = "disable"
	DefaultRedisHost   = "localhost"
	DefaultRedisPort   = "6379"
	DefaultEnvironment = "development"
	DefaultHealthPort  = "8081"
)

type Config struct {
	Environment        string
	HealthPort         string
	EMQXHost           string
	EMQXPort           string
	EMQXUseTLS         bool
	EMQXTLSSkipVerify  bool
	MQTTClientID       string
	MQTTTopic          string
	DatabaseHost       string
	DatabasePort       string
	DatabaseUser       string
	DatabasePassword   string
	DatabaseName       string
	DatabaseSSLMode    string
	RedisHost          string
	RedisPort          string
	RedisPassword      string
}

// LoadConfig loads ingestor configuration from environment.
func LoadConfig() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")

	useTLS := getEnv("EMQX_USE_TLS", "false") == "true" || getEnv("EMQX_PORT", DefaultEMQXPort) == "8883"
	skipVerify := getEnv("EMQX_TLS_SKIP_VERIFY", "true") == "true"

	return &Config{
		Environment:        getEnv("APP_ENV", DefaultEnvironment),
		HealthPort:         getEnv("HEALTH_PORT", DefaultHealthPort),
		EMQXHost:           getEnv("EMQX_HOST", DefaultEMQXHost),
		EMQXPort:           getEnv("EMQX_PORT", DefaultEMQXPort),
		EMQXUseTLS:         useTLS,
		EMQXTLSSkipVerify:  skipVerify,
		MQTTClientID:       getEnv("INGESTOR_MQTT_CLIENT_ID", DefaultClientID),
		MQTTTopic:        getEnv("INGESTOR_MQTT_TOPIC", DefaultMQTTTopic),
		DatabaseHost:     getEnv("DB_HOST", DefaultDBHost),
		DatabasePort:     getEnv("DB_PORT", DefaultDBPort),
		DatabaseUser:     getEnv("DB_USER", DefaultDBUser),
		DatabasePassword: getEnv("DB_PASSWORD", DefaultDBPassword),
		DatabaseName:     getEnv("DB_NAME", DefaultDBName),
		DatabaseSSLMode:  getEnv("DB_SSLMODE", DefaultDBSSLMode),
		RedisHost:        getEnv("REDIS_HOST", DefaultRedisHost),
		RedisPort:        getEnv("REDIS_PORT", DefaultRedisPort),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
