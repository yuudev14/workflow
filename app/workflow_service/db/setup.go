package db

import (
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
)

// Setup DB
func SetupDB(dataSourceName string) (*sqlx.DB, error) {
	var DB *sqlx.DB
	if gin.Mode() == gin.TestMode {
		// Create a new mock database connection
		mockDB, _, mockErr := sqlmock.New()
		if mockErr != nil {
			logging.Sugar.Fatalf("an error '%s' was not expected when opening a stub database connection", mockErr)
			return nil, mockErr
		}
		defer mockDB.Close()

		// Wrap the mock database with sqlx
		DB = sqlx.NewDb(mockDB, "sqlmock")

		return DB, nil

	}
	logging.Sugar.Infof("Connecting to DB... %v", dataSourceName)
	var err error
	DB, err = sqlx.Open("postgres", dataSourceName)

	if err != nil {
		logging.Sugar.Errorf("error opening database: %v", err.Error())
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := DB.Ping(); err != nil {
		logging.Sugar.Errorf("error connecting to database: %v %v", dataSourceName, err.Error())
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return DB, nil
}
