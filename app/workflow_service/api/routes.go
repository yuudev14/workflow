package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	workflow_api "github.com/yuudev14-workflow/workflow-service/api/workflow"
	"github.com/yuudev14-workflow/workflow-service/docs"
	"github.com/yuudev14-workflow/workflow-service/internal/mq"
)

func StartApi(db *sqlx.DB, mqInstance mq.MQStruct, app *gin.RouterGroup) {
	workflow_api.SetupWorkflowController(db, mqInstance, app)
	// auth_api.SetupAuthController(db, app)

}

func InitRouter(db *sqlx.DB, mqInstance mq.MQStruct) *gin.Engine {

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

	StartApi(db, mqInstance, apiGroup)

	app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return app

}
