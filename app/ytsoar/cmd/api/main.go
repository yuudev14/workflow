package main

import (
	"context"
	"errors"
	"log"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/connectorstore"
	"github.com/yuudev14/ytsoar/internal/adapters/depsinstall"
	api "github.com/yuudev14/ytsoar/internal/adapters/http"
	"github.com/yuudev14/ytsoar/internal/adapters/http/handlers"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/adapters/mq"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/adapters/security"
	"github.com/yuudev14/ytsoar/internal/adapters/ws"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
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
	queries := db.New(pool)
	txManager := repository.NewPgxTxManager(pool)

	hub := ws.NewHub(cfg.CORSOrigins)
	go hub.Run()

	mqConn, err := mq.Connect(cfg.MQUrl)
	if err != nil {
		log.Fatalf("failed to connect to MQ: %v", err)
	}
	defer mqConn.Close()

	taskPublisher, err := mq.NewTaskPublisher(appLogger, mqConn, cfg.PlaybookQueueName)
	if err != nil {
		log.Fatalf("failed to setup task publisher: %v", err)
	}

	statusConsumer := mq.NewStatusConsumer(appLogger, mqConn, cfg.StatusExchangeName, hub)
	go func() {
		if err := statusConsumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			appLogger.Errorf("status consumer stopped: %v", err)
		}
	}()

	playbookRepository := repository.NewPlaybookRepository(appLogger, queries, pool)
	taskRepository := repository.NewTaskRepositoryImpl(appLogger, queries, pool)
	edgeRepository := repository.NewEdgeRepositoryImpl(appLogger, queries, pool)
	userRepository := repository.NewUserRepositoryImpl(appLogger, queries, pool)
	roleRepository := repository.NewRoleRepositoryImpl(appLogger, queries, pool)
	refreshTokenRepository := repository.NewRefreshTokenRepositoryImpl(appLogger, queries, pool)
	auditRepository := repository.NewAuditLogRepositoryImpl(appLogger, queries, pool)

	argonHasher := security.NewArgon2Hasher()

	authConfig := auth.AuthConfig{
		JWTSecret:       cfg.JWTSecret,
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
		AdminUsername:   cfg.AdminUsername,
		AdminEmail:      cfg.AdminEmail,
		AdminPassword:   cfg.AdminPassword,
	}

	playbookService := playbooks.NewPlaybookService(appLogger, playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)
	edgeService := edges.NewEdgeServiceImpl(edgeRepository)
	authService := auth.NewService(
		appLogger,
		userRepository,
		roleRepository,
		refreshTokenRepository,
		auditRepository,
		argonHasher,
		txManager,
		authConfig,
	)

	// Without this a fresh database has roles but no users, so nobody could
	// sign in. It is a no-op once someone holds the admin role.
	if err := authService.EnsureAdminUser(ctx); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	orchestrator := playbooks.NewPlaybookApplicationService(
		appLogger,
		playbookService,
		taskService,
		edgeService,
		txManager,
		taskPublisher,
		hub,
	)

	playbookHandler := handlers.NewPlaybookHandler(
		appLogger,
		playbookService,
		taskService,
		edgeService,
		orchestrator,
	)

	cookieWriter := middleware.NewCookieWriter(cfg.CookieSecure)

	authHandler := handlers.NewAuthHandler(
		appLogger,
		authService,
		cookieWriter,
	)

	connectorStore := connectorstore.NewFSStore(appLogger, cfg.ConnectorsDir)
	connectorWriter := connectorstore.NewFSWriter(appLogger, cfg.ConnectorsDir)
	connectorRepository := repository.NewConnectorRepositoryImpl(appLogger, queries, pool)
	connectorInstaller := depsinstall.New(appLogger, cfg.ConnectorsDir)
	connectorService := connectors.NewConnectorService(appLogger, connectorStore,
		connectorWriter, connectorRepository, connectorInstaller)
	connectorHandler := handlers.NewConnectorHandler(appLogger, connectorService)

	routerConfig := api.RouterConfig{
		CORSOrigins: cfg.CORSOrigins,
	}

	app := api.NewRouter(
		routerConfig,
		playbookHandler,
		connectorHandler,
		authHandler,
		hub,
		middleware.Auth(appLogger, authService),
		middleware.AuthFromRefreshCookie(appLogger, authService),
		middleware.NewPermissionMiddleware(appLogger, authService),
	)
	if err := app.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
