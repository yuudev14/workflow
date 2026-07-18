package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultNodeMemoryLimitMB   = 256
	DefaultPythonMemoryLimitMB = 512
)

type Config struct {
	DBUrl             string
	LoggerMode        string
	MQUrl             string
	PlaybookQueueName string
	JWTSecret         string
	HTTPAddr          string

	StatusExchangeName string

	// SandboxAddr is where the worker sends every dynamic node
	// (ConnectorRuntime gRPC on cmd/sandbox).
	SandboxAddr string

	// SandboxListenAddr is the sandbox's own listen address.
	SandboxListenAddr string

	// ConnectorsDir is the unified connectors tree (a directory literally
	// named "connectors": <id>/{info.json, connector.py|connector.ts|
	// connector.js, configs/}). The API serves metadata from it; the sandbox
	// executes from it. Empty disables connectors.
	ConnectorsDir string

	// Worker execution tuning.
	MaxParallelNodes int           // nodes run concurrently within one playbook run
	PlaybookPrefetch int           // playbook messages one worker runs concurrently (MQ Qos)
	NodeTimeout      time.Duration // per-node execution timeout (sandbox fallback too)

	// Sandbox subprocess memory caps. Keep concurrency x cap under the
	// sandbox container's memory limit.
	NodeMemoryLimitMB   int // V8 --max-old-space-size for JS/TS subprocesses
	PythonMemoryLimitMB int // RLIMIT_AS the python harnesses apply to themselves
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

		StatusExchangeName: getEnvOr("STATUS_EXCHANGE", "playbook.status"),

		SandboxAddr:       getEnvOr("SANDBOX_ADDR", "localhost:50052"),
		SandboxListenAddr: getEnvOr("SANDBOX_LISTEN_ADDR", ":50052"),
		ConnectorsDir:     getEnvOr("CONNECTORS_DIR", ""),

		MaxParallelNodes: getEnvIntOr("MAX_PARALLEL_NODES", 4),
		PlaybookPrefetch: getEnvIntOr("PLAYBOOK_PREFETCH", 1),
		NodeTimeout:      time.Duration(getEnvIntOr("NODE_TIMEOUT_SECONDS", 300)) * time.Second,

		NodeMemoryLimitMB:   getEnvIntOr("NODE_MEMORY_LIMIT_MB", DefaultNodeMemoryLimitMB),
		PythonMemoryLimitMB: getEnvIntOr("PYTHON_MEMORY_LIMIT_MB", DefaultPythonMemoryLimitMB),
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

// getEnvIntOr parses an integer env var; unset or unparsable values fall back.
func getEnvIntOr(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
