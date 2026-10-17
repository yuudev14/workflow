package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/connectorstore"
	"github.com/yuudev14/ytsoar/internal/adapters/depsinstall"
	"github.com/yuudev14/ytsoar/internal/adapters/grpcserver"
	api "github.com/yuudev14/ytsoar/internal/adapters/http"
	"github.com/yuudev14/ytsoar/internal/adapters/http/handlers"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/adapters/jobs"
	"github.com/yuudev14/ytsoar/internal/adapters/oidcclient"
	"github.com/yuudev14/ytsoar/internal/adapters/pgstatus"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/goconnectors"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/grpcruntime"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/localexec"
	"github.com/yuudev14/ytsoar/internal/adapters/security"
	"github.com/yuudev14/ytsoar/internal/adapters/templating"
	"github.com/yuudev14/ytsoar/internal/adapters/ws"
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// NewAPI connects the adapters and services of the api program and returns its HTTP
// router. cmd/api and the integration tests both call it, so tests use the real wiring.
func NewAPI(ctx context.Context, cfg config.Config, appLogger logger.Logger, pool *pgxpool.Pool) (*gin.Engine, error) {
	queries := db.New(pool)
	txManager := repository.NewPgxTxManager(pool)

	hub := ws.NewHub(cfg.CORSOrigins)
	go hub.Run()

	runEnqueuer, err := jobs.NewEnqueuer(pool, repository.TxFromContext)
	if err != nil {
		return nil, fmt.Errorf("failed to setup job queue: %w", err)
	}

	playbookRepository := repository.NewPlaybookRepository(appLogger, queries, pool)
	taskRepository := repository.NewTaskRepositoryImpl(appLogger, queries, pool)
	edgeRepository := repository.NewEdgeRepositoryImpl(appLogger, queries, pool)
	userRepository := repository.NewUserRepositoryImpl(appLogger, queries, pool)
	roleRepository := repository.NewRoleRepositoryImpl(appLogger, queries, pool)
	refreshTokenRepository := repository.NewRefreshTokenRepositoryImpl(appLogger, queries, pool)
	teamRepository := repository.NewTeamRepositoryImpl(appLogger, queries, pool)
	auditRepository := repository.NewAuditLogRepositoryImpl(appLogger, queries, pool)
	authProviderRepository := repository.NewAuthProviderRepositoryImpl(appLogger, queries, pool)

	argonHasher := security.NewArgon2Hasher()
	oidcClient := oidcclient.New()

	authConfig := auth.AuthConfig{
		JWTSecret:       cfg.JWTSecret,
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
		AdminUsername:   cfg.AdminUsername,
		AdminEmail:      cfg.AdminEmail,
		AdminPassword:   cfg.AdminPassword,
		FrontendURL:     cfg.FrontendURL,
		AuthPublicURL:   cfg.AuthPublicURL,
	}

	playbookService := playbooks.NewPlaybookService(appLogger, playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)

	statusListener := pgstatus.NewListener(appLogger, pool, playbookService, taskService, hub)
	go func() {
		if err := statusListener.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			appLogger.Errorf("status listener stopped: %v", err)
		}
	}()
	edgeService := edges.NewEdgeServiceImpl(edgeRepository)
	authService := auth.NewService(
		appLogger,
		userRepository,
		roleRepository,
		refreshTokenRepository,
		teamRepository,
		auditRepository,
		authProviderRepository,
		argonHasher,
		oidcClient,
		txManager,
		authConfig,
	)

	// without this a new database has roles but no users, and nobody can sign in.
	// It does nothing once someone has the admin role.
	if err := authService.EnsureAdminUser(ctx); err != nil {
		return nil, fmt.Errorf("failed to seed admin user: %w", err)
	}

	alertRepository := repository.NewAlertRepositoryImpl(appLogger, queries, pool)
	incidentRepository := repository.NewIncidentRepositoryImpl(appLogger, queries, pool)

	orchestrator := playbooks.NewPlaybookApplicationService(
		appLogger,
		playbookService,
		taskService,
		edgeService,
		txManager,
		runEnqueuer,
		hub,
		repository.NewRecordResolverImpl(appLogger, alertRepository, incidentRepository),
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
	adminHandler := handlers.NewAdminHandler(appLogger, authService)

	moduleEventPublisher := jobs.NewModuleEventLogger(appLogger)

	// the alert repository is also incidents.AlertTimeline, so a link writes to the alert too
	incidentService := incidents.NewService(
		appLogger, incidentRepository, alertRepository, txManager, moduleEventPublisher, userRepository)
	alertService := alerts.NewService(
		appLogger, alertRepository, incidentService, txManager, moduleEventPublisher, userRepository)

	alertHandler := handlers.NewAlertHandler(appLogger, alertService, orchestrator)
	incidentHandler := handlers.NewIncidentHandler(appLogger, incidentService, orchestrator)

	routerConfig := api.RouterConfig{
		CORSOrigins: cfg.CORSOrigins,
	}

	return api.NewRouter(
		routerConfig,
		playbookHandler,
		connectorHandler,
		authHandler,
		adminHandler,
		alertHandler,
		incidentHandler,
		hub,
		middleware.Auth(appLogger, authService),
		middleware.AuthFromRefreshCookie(appLogger, authService),
		middleware.NewPermissionMiddleware(appLogger, authService),
	), nil
}

// NewWorker connects the parts of the worker program and returns its job queue client.
// The caller starts and stops it.
func NewWorker(cfg config.Config, appLogger logger.Logger, pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	queries := db.New(pool)

	playbookRepository := repository.NewPlaybookRepository(appLogger, queries, pool)
	taskRepository := repository.NewTaskRepositoryImpl(appLogger, queries, pool)

	playbookService := playbooks.NewPlaybookService(appLogger, playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)

	// the worker never runs user code. SandboxRuntime sends all other tasks (python/js/ts
	// connectors, code snippets) to the sandbox over gRPC. The built-in connectors are our
	// code, so BuiltinRuntime runs them in the worker.
	sandboxRuntime, err := grpcruntime.NewSandboxRuntime(appLogger, cfg.SandboxAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to setup sandbox runtime client: %w", err)
	}

	builtinRuntime := goconnectors.NewBuiltinRuntime(appLogger, templating.NewGonjaEngine(), cfg.ConnectorsDir)
	builtinRuntime.Register("http_request", goconnectors.NewHTTPRequestConnector())
	byConnector := map[string]execution.TaskRuntime{}
	for _, id := range builtinRuntime.IDs() {
		byConnector[id] = builtinRuntime
	}
	resolver := execution.NewStaticResolver(sandboxRuntime, byConnector)

	executor := execution.NewExecutor(
		appLogger,
		taskService,
		playbookService,
		resolver,
		pgstatus.NewNotifier(pool),
		cfg.MaxParallelNodes,
		cfg.NodeTimeout,
	)
	playbookRunLoader := playbooks.NewRunLoader(playbookService, taskService)
	runWorker := jobs.NewRunWorker(appLogger, playbookRunLoader, executor)

	return jobs.NewWorkerClient(
		pool,
		runWorker,
		cfg.PlaybookConcurrency, cfg.RunTimeout, cfg.WorkerStopGrace)
}

// NewSandbox connects the parts of the sandbox program and returns its gRPC server.
// The sandbox has no database access.
func NewSandbox(cfg config.Config, appLogger logger.Logger) (*grpcserver.RuntimeServer, error) {
	// code nodes are pinned first, connector ids from the unified tree never override them
	byConnector := map[string]execution.TaskRuntime{
		"code_snippet_py": localexec.NewPythonSnippetRuntime(appLogger, cfg.PythonMemoryLimitMB),
		"code_snippet_js": localexec.NewNodeJSSnippetRuntime(appLogger, cfg.NodeMemoryLimitMB),
	}

	// python connectors are the default: a connector without a mapping uses the python
	// harness. A connector with "runtime": "node" in info.json uses the node harness. This is
	// checked per request, so a connector uploaded after boot works without a restart.
	var defaultRuntime execution.TaskRuntime
	var nodeJSConnectorRuntime execution.TaskRuntime
	if cfg.ConnectorsDir != "" {
		pythonConnectorRuntime, err := localexec.NewPythonConnectorRuntime(appLogger, cfg.ConnectorsDir, cfg.PythonMemoryLimitMB)
		if err != nil {
			return nil, fmt.Errorf("failed to setup python connector runtime: %w", err)
		}
		defaultRuntime = pythonConnectorRuntime

		nodeJSRuntime, err := localexec.NewNodeJSConnectorRuntime(appLogger, cfg.ConnectorsDir, cfg.NodeMemoryLimitMB)
		if err != nil {
			return nil, fmt.Errorf("failed to setup js connector runtime: %w", err)
		}
		nodeJSConnectorRuntime = nodeJSRuntime
	} else {
		appLogger.Warn("CONNECTORS_DIR is empty: connectors are disabled, only code nodes will run")
	}

	// DynamicResolver runs in the sandbox. It picks the runtime of each task from the worker
	// and reads info.json on every call, so an uploaded connector works without a restart.
	resolver := execution.DynamicResolver(func(task domain.Tasks) (execution.TaskRuntime, error) {
		if task.ConnectorID != nil {
			if runtime, ok := byConnector[*task.ConnectorID]; ok {
				return runtime, nil
			}
			if nodeJSConnectorRuntime != nil && localexec.IsNodeJSConnector(cfg.ConnectorsDir, *task.ConnectorID) {
				return nodeJSConnectorRuntime, nil
			}
		}
		if defaultRuntime == nil {
			return nil, fmt.Errorf("no runtime registered for connector %v", task.ConnectorID)
		}
		return defaultRuntime, nil
	})
	return grpcserver.NewRuntimeServer(appLogger, resolver, cfg.NodeTimeout), nil

}
