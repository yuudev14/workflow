package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/yuudev14/ytsoar/docs"
	"github.com/yuudev14/ytsoar/internal/adapters/http/handlers"
	"github.com/yuudev14/ytsoar/internal/adapters/ws"
)

func NewRouter(playbookHandler *handlers.PlaybookHandler, hub *ws.Hub) *gin.Engine {
	app := gin.Default()

	config := cors.Config{
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}
	app.Use(cors.New(config))

	docs.SwaggerInfo.BasePath = "./"

	apiGroup := app.Group("/api")
	playbookHandler.RegisterRoutes(apiGroup)

	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	app.GET("/ws/playbook", hub.ServeWS)

	return app
}
