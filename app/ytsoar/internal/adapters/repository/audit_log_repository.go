package repository

import (
	"context"
	"encoding/json"

	sq "github.com/Masterminds/squirrel"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// AuditLogRepositoryImpl implements auth.AuditLogRepository.
type AuditLogRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewAuditLogRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *AuditLogRepositoryImpl {
	return &AuditLogRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *AuditLogRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func applyAuditFilter(stmt sq.SelectBuilder, filter auth.AuditFilter) sq.SelectBuilder {
	if filter.ActorID != nil {
		stmt = stmt.Where(sq.Eq{"a.actor_id": *filter.ActorID})
	}
	if filter.Module != nil {
		stmt = stmt.Where(sq.Eq{"a.module": *filter.Module})
	}
	if filter.Action != nil {
		stmt = stmt.Where(sq.Eq{"a.action": *filter.Action})
	}
	if filter.From != nil {
		stmt = stmt.Where(sq.GtOrEq{"a.created_at": *filter.From})
	}
	if filter.To != nil {
		stmt = stmt.Where(sq.LtOrEq{"a.created_at": *filter.To})
	}
	return stmt
}

// The join is LEFT: actor_id is ON DELETE SET NULL, and a row whose actor was
// removed must still appear in the trail.
func selectAuditLogs(columns string) sq.SelectBuilder {
	return sq.Select(columns).
		From("audit_logs a").
		LeftJoin("users u ON u.id = a.actor_id").
		PlaceholderFormat(sq.Dollar)
}

func (r *AuditLogRepositoryImpl) List(ctx context.Context, offset, limit int, filter auth.AuditFilter) ([]domain.AuditLog, error) {
	stmt := applyAuditFilter(selectAuditLogs("a.*, u.username AS actor_username"), filter).
		OrderBy("a.created_at DESC").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	return CollectRowsFromSqlizer[domain.AuditLog](ctx, stmt, r.pool, r.logger)
}

func (r *AuditLogRepositoryImpl) Count(ctx context.Context, filter auth.AuditFilter) (int, error) {
	stmt := applyAuditFilter(selectAuditLogs("count(*)"), filter)
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

func (r *AuditLogRepositoryImpl) Insert(ctx context.Context, entry domain.AuditEntry) error {
	var detail json.RawMessage
	if entry.Detail != nil {
		encoded, err := json.Marshal(entry.Detail)
		if err != nil {
			return err
		}
		detail = encoded
	}

	actorID := pgtype.UUID{}
	if entry.ActorID != nil {
		actorID = toPgUUID(*entry.ActorID)
	}

	_, err := r.queriesFromContext(ctx).InsertAuditLog(ctx, db.InsertAuditLogParams{
		ActorID:  actorID,
		Module:   entry.Module,
		Action:   entry.Action,
		EntityID: toPgText(entry.EntityID),
		Detail:   detail,
	})
	return err
}
