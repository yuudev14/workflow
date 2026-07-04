// Package logger configures the application logger.
package logger

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:generate mockgen -destination=mocks/mock_logger.go -package=mocks . Logger
type Logger interface {
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Debugw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)

	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Debug(args ...any)
	Fatal(args ...any)

	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)
	Debugf(template string, args ...any)
	Fatalf(template string, args ...any)

	Sync() error
}

func devColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	const (
		violet = "\x1b[35m"
		blue   = "\x1b[34m"
		yellow = "\x1b[33m"
		red    = "\x1b[31m"
		reset  = "\x1b[0m"
	)
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString(violet + "DEBUG" + reset)
	case zapcore.InfoLevel:
		enc.AppendString(blue + "INFO" + reset)
	case zapcore.WarnLevel:
		enc.AppendString(yellow + "WARN" + reset)
	case zapcore.ErrorLevel:
		enc.AppendString(red + "ERROR" + reset)
	default:
		enc.AppendString(red + l.CapitalString() + reset)
	}
}

func SetupLogger() Logger {
	var logger *zap.Logger
	var err error

	if os.Getenv("APP_ENV") == "production" {
		logger, err = zap.NewProduction()
	} else {
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = devColorLevelEncoder
		logger, err = cfg.Build()
	}
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}

	return logger.Sugar()
}

// NewNop returns a logger that discards everything; useful in tests.
func NewNop() Logger {
	return zap.NewNop().Sugar()
}
