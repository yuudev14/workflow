package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	auth_api "github.com/yuudev14-workflow/workflow-service/api/auth"
	workflow_api "github.com/yuudev14-workflow/workflow-service/api/workflow"
	"github.com/yuudev14-workflow/workflow-service/docs"
)

func StartApi(app *gin.RouterGroup) {
	workflow_api.SetupWorkflowController(app)
	auth_api.SetupAuthController(app)

}

func InitRouter() *gin.Engine {

	app := gin.Default()

	config := cors.Config{
		//AllowOrigins:    []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
	}

	// Use CORS middleware
	app.Use(cors.New(config))

	docs.SwaggerInfo.BasePath = "./"

	apiGroup := app.Group("/api")

	StartApi(apiGroup)

	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return app

}
