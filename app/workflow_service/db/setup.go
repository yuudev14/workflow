package db

import (
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
)

var DB *sqlx.DB

// Setup DB
func SetupDB(dataSourceName string) error {
	if gin.Mode() == gin.TestMode {
		// Create a new mock database connection
		mockDB, _, mockErr := sqlmock.New()
		if mockErr != nil {
			logging.Sugar.Fatalf("an error '%s' was not expected when opening a stub database connection", mockErr)
			return mockErr
		}
		defer mockDB.Close()

		// Wrap the mock database with sqlx
		DB = sqlx.NewDb(mockDB, "sqlmock")

		return nil

	}
	logging.Sugar.Infof("Connecting to DB... %v", dataSourceName)
	var err error
	DB, err = sqlx.Open("postgres", dataSourceName)

	if err != nil {
		logging.Sugar.Errorf("error opening database: %v", err.Error())
		return fmt.Errorf("error opening database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		logging.Sugar.Errorf("error connecting to database: %v %v", dataSourceName, err.Error())
		return fmt.Errorf("error connecting to database: %w", err)
	}
	return err
}
