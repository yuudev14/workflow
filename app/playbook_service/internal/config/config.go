package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl             string
	LoggerMode        string
	MQUrl             string
	PlaybookQueueName string
	JWTSecret         string
	HTTPAddr          string
	GRPCAddr          string
}

// Load reads .env (when present) and assembles the root configuration.
func Load() Config {
	return LoadFrom("")
}

// LoadFrom loads configuration using the given dotenv file path. An empty
// path falls back to ./.env.
func LoadFrom(dest string) Config {
	if dest == "" {
		godotenv.Load("./.env")
	} else {
		godotenv.Load(dest)
	}

	return Config{
		DBUrl:             getDbUrl(),
		LoggerMode:        getEnvOr("LOGGER_MODE", "DEBUG"),
		MQUrl:             getMQUrl(),
		PlaybookQueueName: getEnvOr("PLAYBOOK_QUEUE", "playbook"),
		JWTSecret:         getEnvOr("JWT_SECRET", "secret"),
		HTTPAddr:          getEnvOr("HTTP_ADDR", ":8080"),
		GRPCAddr:          getEnvOr("GRPC_ADDR", ":50051"),
	}
}

func getDbUrl() string {
	return fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
}

func getMQUrl() string {
	return fmt.Sprintf(
		"amqp://%v:%v@%v:%v/",
		os.Getenv("MQ_USER"),
		os.Getenv("MQ_PASSWORD"),
		os.Getenv("MQ_HOST"),
		os.Getenv("MQ_PORT"),
	)
}

func getEnvOr(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
