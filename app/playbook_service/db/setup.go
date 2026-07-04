package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/yuudev14/ytsoar/internal/logging"
)

// SetupDB opens and pings the Postgres connection pool.
func SetupDB(dataSourceName string) (*sqlx.DB, error) {
	logging.Sugar.Infof("Connecting to DB... %v", dataSourceName)
	DB, err := sqlx.Open("postgres", dataSourceName)
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
