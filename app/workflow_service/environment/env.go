package environment

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

var (
	Settings = &SettingsData{}

	// local variables
	loggerMode = "DEBUG"
)

type SettingsData struct {
	DB_URL            string
	LOGGER_MODE       string
	MQ_URL            string
	ReceiverQueueName string
	SenderQueueName   string
	JWT_SECRET        string
}

// load env based
func LoadEnv(dest string) {
	if dest == "" {
		godotenv.Load("./.env")
	} else {
		godotenv.Load(dest)
	}
}

// set constants environment
func SetEnv() {
	Settings.DB_URL = GetDbUrl()
	Settings.LOGGER_MODE = GetLoggerLevel()
	Settings.MQ_URL = GetMQUrl()
	Settings.SenderQueueName = "workflow"
	Settings.ReceiverQueueName = "workflow_processor"
}

// function to retrieve contants in the env
func Setup() {
	LoadEnv("")
	SetEnv()
}

func TestSetup(envDest string) {
	LoadEnv(envDest)
	SetEnv()
}

func GetDbUrl() string {
	// postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}/${DB_NAME}?sslmode=disable
	DB_USER := os.Getenv("DB_USER")
	DB_PASSWORD := os.Getenv("DB_PASSWORD")
	DB_HOST := os.Getenv("DB_HOST")
	DB_NAME := os.Getenv("DB_NAME")
	return fmt.Sprintf("postgres://%v:%v@%v/%v?sslmode=disable", DB_USER, DB_PASSWORD, DB_HOST, DB_NAME)
}

func GetMQUrl() string {
	// amqp://guest:guest@localhost:5672/
	MQ_USER := os.Getenv("MQ_USER")
	MQ_PASSWORD := os.Getenv("MQ_PASSWORD")
	MQ_HOST := os.Getenv("MQ_HOST")
	MQ_PORT := os.Getenv("MQ_PORT")
	return fmt.Sprintf("amqp://%v:%v@%v:%v/", MQ_USER, MQ_PASSWORD, MQ_HOST, MQ_PORT)
}

// get logger level
func GetLoggerLevel() string {
	val := os.Getenv("LOGGER_MODE")
	if val == "" {
		return loggerMode
	}
	return val

}

// get logger level
func GetJwtSecret() string {
	val := os.Getenv("JWT_SECRET")
	if val == "" {
		return "secret"
	}
	return val

}
