package main

import (
	"context"
	"log"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/app"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// @title 	YTSoar Playbook Service API
// @version	1.0
// @description YTSoar playbook service in Go using the Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	if cfg.AppEnv == "production" && cfg.JWTSecret == config.DefaultJWTSecret {
		log.Fatal("JWT_SECRET must be set to a private value when APP_ENV=production")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}
	defer pool.Close()

	app, err := app.NewAPI(ctx, cfg, appLogger, pool)
	if err != nil {
		log.Fatal(err)
	}
	if err := app.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
