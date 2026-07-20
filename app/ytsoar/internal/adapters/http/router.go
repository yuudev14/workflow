package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/yuudev14/ytsoar/docs"
	"github.com/yuudev14/ytsoar/internal/adapters/http/handlers"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/adapters/ws"
	"github.com/yuudev14/ytsoar/internal/domain"
)

type RouterConfig struct {
	// CORSOrigins is an exact allow-list. A wildcard is not an option here:
	// browsers refuse "*" once credentials are involved, and the refresh
	// cookie makes every request credentialed.
	CORSOrigins []string
}

func NewRouter(
	cfg RouterConfig,
	playbookHandler *handlers.PlaybookHandler,
	connectorHandler *handlers.ConnectorHandler,
	authHandler *handlers.AuthHandler,
	adminHandler *handlers.AdminHandler,
	hub *ws.Hub,
	authMW gin.HandlerFunc,
	wsAuthMW gin.HandlerFunc,
	requirePermission middleware.PermissionMiddleware,
) *gin.Engine {
	app := gin.Default()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	docs.SwaggerInfo.BasePath = "./"

	apiGroup := app.Group("/api")

	// Login, refresh and logout have to work without an access token.
	authHandler.RegisterPublicRoutes(apiGroup)

	protected := apiGroup.Group("", authMW)

	// /me carries no grant: every authenticated user must be able to read
	// their own profile and permissions.
	authHandler.RegisterProtectedRoutes(protected)

	playbookHandler.RegisterRoutes(protected, requirePermission)
	connectorHandler.RegisterRoutes(protected, requirePermission)
	adminHandler.RegisterRoutes(protected, requirePermission)

	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// The handshake authenticates with the refresh cookie: a browser cannot
	// set an Authorization header on a WebSocket. The socket carries playbook
	// status, so it needs the same grant as reading playbooks.
	app.GET("/ws/playbook",
		wsAuthMW,
		requirePermission(domain.ModulePlaybooks, domain.ActionRead),
		hub.ServeWS)

	return app
}
