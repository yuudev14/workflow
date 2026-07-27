package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestMapUniqueViolation(t *testing.T) {
	conflict := errors.New("name already taken")
	unique := &pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: "auth_providers_name_key"}

	assert.ErrorIs(t, mapUniqueViolation(unique, conflict), conflict)
	// the driver error is often wrapped by the time it reaches a repository.
	assert.ErrorIs(t, mapUniqueViolation(fmt.Errorf("insert: %w", unique), conflict), conflict)

	// anything else must pass through untouched - a foreign-key breach or a dead
	// connection is not a conflict and must not be reported as one.
	fk := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}
	assert.ErrorIs(t, mapUniqueViolation(fk, conflict), fk)
	assert.ErrorIs(t, mapUniqueViolation(pgx.ErrNoRows, conflict), pgx.ErrNoRows)
	assert.NoError(t, mapUniqueViolation(nil, conflict))
}
