package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

// CollectRowsFromSqlizer builds the squirrel statement and maps every row
// onto T by column name (db struct tags).
func CollectRowsFromSqlizer[T any](
	ctx context.Context,
	stmt sq.Sqlizer,
	pool *pgxpool.Pool,
	log logger.Logger,
) ([]T, error) {
	sqlStr, args, err := stmt.ToSql()
	if err != nil {
		return nil, err
	}

	log.Debugw("sql collect rows", "sql", sqlStr, "args", args)
	rows, err := pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}

// CollectOneScalarFromSqlizer builds the squirrel statement and scans a
// single value (e.g. count(*)).
func CollectOneScalarFromSqlizer[T any](
	ctx context.Context,
	stmt sq.Sqlizer,
	pool *pgxpool.Pool,
	log logger.Logger,
) (T, error) {
	var v T
	sqlStr, args, err := stmt.ToSql()
	if err != nil {
		return v, err
	}

	log.Debugw("sql collect scalar", "sql", sqlStr, "args", args)
	if err := pool.QueryRow(ctx, sqlStr, args...).Scan(&v); err != nil {
		return v, err
	}
	return v, nil
}

// mapNoRows keeps "no such row" distinguishable from a database failure, so
// callers can 404 one and 500 the other.
func mapNoRows(err error, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDFromString(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return toPgUUID(parsed), nil
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func toPgTextFromString(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func fromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func fromPgTextString(t pgtype.Text) string {
	return t.String
}

func toPgTextFromNullable(n types.Nullable[string]) pgtype.Text {
	if !n.Set || n.Value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *n.Value, Valid: true}
}

func toPgBoolFromNullable(n types.Nullable[bool]) pgtype.Bool {
	if !n.Set || n.Value == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *n.Value, Valid: true}
}

func toNullString(t pgtype.Text) sql.NullString {
	return sql.NullString{String: t.String, Valid: t.Valid}
}

func toPgTextFromNullString(n sql.NullString) pgtype.Text {
	return pgtype.Text{String: n.String, Valid: n.Valid}
}

func fromPgTimestampPtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toPgFloat8(f float32) pgtype.Float8 {
	return pgtype.Float8{Float64: float64(f), Valid: true}
}

func toNullPlaybookStatus(n types.Nullable[string]) db.NullPlaybookStatus {
	if !n.Set || n.Value == nil {
		return db.NullPlaybookStatus{}
	}
	return db.NullPlaybookStatus{PlaybookStatus: db.PlaybookStatus(*n.Value), Valid: true}
}

func toNullTaskStatus(n types.Nullable[string]) db.NullTaskStatus {
	if !n.Set || n.Value == nil {
		return db.NullTaskStatus{}
	}
	return db.NullTaskStatus{TaskStatus: db.TaskStatus(*n.Value), Valid: true}
}

func toNullTriggerTypePtr(s *string) db.NullTriggerType {
	if s == nil {
		return db.NullTriggerType{}
	}
	return db.NullTriggerType{TriggerType: db.TriggerType(*s), Valid: true}
}

func toNullTriggerTypeFromNullable(n types.Nullable[string]) db.NullTriggerType {
	if !n.Set || n.Value == nil {
		return db.NullTriggerType{}
	}
	return db.NullTriggerType{TriggerType: db.TriggerType(*n.Value), Valid: true}
}

func fromNullTriggerType(nt db.NullTriggerType) *string {
	if !nt.Valid {
		return nil
	}
	s := string(nt.TriggerType)
	return &s
}
