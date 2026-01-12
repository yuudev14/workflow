package utils

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
)

func NullStringToInterface(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func SetupMockEnvironment(t *testing.T) (*sql.DB, *sqlx.DB, sqlmock.Sqlmock) {
	environment.Setup()
	logging.Setup("DEBUG")

	// Create a new mock database connection
	mockDB, sqlmock, mockErr := sqlmock.New()

	if mockErr != nil {
		logging.Sugar.Fatalf("an error '%s' was not expected when opening a stub database connection", mockErr)
	}
	t.Cleanup(func() {
		mockDB.Close()
		sqlmock.ExpectationsWereMet()
	})

	// Wrap the mock database with sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	t.Cleanup(func() {
		sqlxDB.Close()
	})

	return mockDB, sqlxDB, sqlmock

}

// Helper
func StrPtr(s string) *string { return &s }
