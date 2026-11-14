package repository

import (
	"context"
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/schedules"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type ScheduleRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewScheduleRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *ScheduleRepositoryImpl {
	return &ScheduleRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *ScheduleRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func (r *ScheduleRepositoryImpl) Create(ctx context.Context, s domain.Schedule) (domain.Schedule, error) {
	params, err := json.Marshal(s.Parameters)
	if err != nil {
		return domain.Schedule{}, err
	}
	id, err := r.queriesFromContext(ctx).CreateSchedule(ctx, db.CreateScheduleParams{
		Name:       s.Name,
		PlaybookID: toPgUUID(s.PlaybookID),
		Cron:       s.Cron,
		Timezone:   s.Timezone,
		Enabled:    s.Enabled,
		Parameters: params,
		NextRunAt:  toPgTimestamptz(s.NextRunAt),
		CreatedBy:  toPgUUIDPtr(s.CreatedBy),
	})
	if err != nil {
		return domain.Schedule{}, err
	}
	return r.Get(ctx, fromPgUUID(id))
}

func (r *ScheduleRepositoryImpl) Get(ctx context.Context, id uuid.UUID) (domain.Schedule, error) {
	row, err := r.queriesFromContext(ctx).GetSchedule(ctx, toPgUUID(id))
	if err != nil {
		return domain.Schedule{}, mapNoRows(err, schedules.ErrScheduleNotFound)
	}
	return toDomainSchedule(row), nil
}

func scheduleFilter(stmt sq.SelectBuilder, playbookID *uuid.UUID) sq.SelectBuilder {
	if playbookID != nil {
		stmt = stmt.Where(sq.Eq{"s.playbook_id": *playbookID})
	}
	return stmt
}

func (r *ScheduleRepositoryImpl) Count(ctx context.Context, playbookID *uuid.UUID) (int, error) {
	stmt := scheduleFilter(sq.Select("count(*)").From("schedulers s").PlaceholderFormat(sq.Dollar), playbookID)
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

// scheduleRow has the columns of the list query, matched by name
type scheduleRow struct {
	ID                  uuid.UUID       `db:"id"`
	Name                string          `db:"name"`
	PlaybookID          uuid.UUID       `db:"playbook_id"`
	PlaybookName        *string         `db:"playbook_name"`
	PlaybookTriggerType *string         `db:"playbook_trigger_type"`
	PlaybookDeleted     bool            `db:"playbook_deleted"`
	Cron                string          `db:"cron"`
	Timezone            string          `db:"timezone"`
	Enabled             bool            `db:"enabled"`
	Parameters          json.RawMessage `db:"parameters"`
	NextRunAt           time.Time       `db:"next_run_at"`
	LastRunAt           *time.Time      `db:"last_run_at"`
	CreatedBy           *uuid.UUID      `db:"created_by"`
	CreatedAt           time.Time       `db:"created_at"`
	UpdatedAt           time.Time       `db:"updated_at"`
}

const scheduleListColumns = `s.id, s.name, s.playbook_id, p.name AS playbook_name,
    p.trigger_type::text AS playbook_trigger_type, (p.deleted_at IS NOT NULL) AS playbook_deleted,
    s.cron, s.timezone, s.enabled, s.parameters, s.next_run_at, s.last_run_at, s.created_by,
    s.created_at, s.updated_at`

func (r *ScheduleRepositoryImpl) List(ctx context.Context, playbookID *uuid.UUID, offset, limit int) ([]domain.Schedule, error) {
	query := sq.Select(scheduleListColumns).
		From("schedulers s").
		Join("playbooks p ON p.id = s.playbook_id").
		PlaceholderFormat(sq.Dollar)
	stmt := scheduleFilter(query, playbookID).
		OrderBy("s.name", "s.id").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	rows, err := CollectRowsFromSqlizer[scheduleRow](ctx, stmt, r.pool, r.logger)
	if err != nil {
		return nil, err
	}
	list := make([]domain.Schedule, len(rows))
	for i, row := range rows {
		params := map[string]any{}
		_ = json.Unmarshal(row.Parameters, &params)
		name := ""
		if row.PlaybookName != nil {
			name = *row.PlaybookName
		}
		list[i] = domain.Schedule{
			ID:                  row.ID,
			Name:                row.Name,
			PlaybookID:          row.PlaybookID,
			PlaybookName:        name,
			PlaybookTriggerType: row.PlaybookTriggerType,
			PlaybookDeleted:     row.PlaybookDeleted,
			Cron:                row.Cron,
			Timezone:            row.Timezone,
			Enabled:             row.Enabled,
			Parameters:          params,
			NextRunAt:           row.NextRunAt,
			LastRunAt:           row.LastRunAt,
			CreatedBy:           row.CreatedBy,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
		}
	}
	return list, nil
}

func (r *ScheduleRepositoryImpl) Update(ctx context.Context, s domain.Schedule) (domain.Schedule, error) {
	params, err := json.Marshal(s.Parameters)
	if err != nil {
		return domain.Schedule{}, err
	}
	id, err := r.queriesFromContext(ctx).UpdateSchedule(ctx, db.UpdateScheduleParams{
		ID:         toPgUUID(s.ID),
		Name:       s.Name,
		Cron:       s.Cron,
		Timezone:   s.Timezone,
		Enabled:    s.Enabled,
		Parameters: params,
		NextRunAt:  toPgTimestamptz(s.NextRunAt),
	})
	if err != nil {
		return domain.Schedule{}, mapNoRows(err, schedules.ErrScheduleNotFound)
	}
	return r.Get(ctx, fromPgUUID(id))
}

func (r *ScheduleRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.queriesFromContext(ctx).DeleteSchedule(ctx, toPgUUID(id))
	if err != nil {
		return err
	}
	if n == 0 {
		return schedules.ErrScheduleNotFound
	}
	return nil
}

func (r *ScheduleRepositoryImpl) ListDueIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	rows, err := r.queriesFromContext(ctx).ListDueScheduleIDs(ctx, toPgTimestamptz(now))
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = fromPgUUID(row)
	}
	return ids, nil
}

func (r *ScheduleRepositoryImpl) LockDue(ctx context.Context, id uuid.UUID, now time.Time) (domain.Schedule, bool, error) {
	row, err := r.queriesFromContext(ctx).LockDueSchedule(ctx, db.LockDueScheduleParams{
		ID: toPgUUID(id), NextRunAt: toPgTimestamptz(now),
	})
	if err != nil {
		if mapNoRows(err, nil) == nil {
			return domain.Schedule{}, false, nil
		}
		return domain.Schedule{}, false, err
	}
	return toDomainSchedule(db.GetScheduleRow{
		ID:         row.ID,
		Name:       row.Name,
		PlaybookID: row.PlaybookID,
		Cron:       row.Cron,
		Timezone:   row.Timezone,
		Enabled:    row.Enabled,
		Parameters: row.Parameters,
		NextRunAt:  row.NextRunAt,
		LastRunAt:  row.LastRunAt,
		CreatedBy:  row.CreatedBy,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}), true, nil
}

func (r *ScheduleRepositoryImpl) SetNextRun(ctx context.Context, id uuid.UUID, lastRunAt *time.Time, next time.Time) error {
	arg := db.SetScheduleNextRunParams{ID: toPgUUID(id), NextRunAt: toPgTimestamptz(next)}
	if lastRunAt != nil {
		arg.LastRunAt = toPgTimestamptz(*lastRunAt)
	}
	return r.queriesFromContext(ctx).SetScheduleNextRun(ctx, arg)
}

func toDomainSchedule(row db.GetScheduleRow) domain.Schedule {
	var trigger *string
	if row.PlaybookTriggerType != "" {
		trigger = &row.PlaybookTriggerType
	}
	params := map[string]any{}
	_ = json.Unmarshal(row.Parameters, &params)
	name := ""
	if n := fromPgText(row.PlaybookName); n != nil {
		name = *n
	}
	return domain.Schedule{
		ID:                  fromPgUUID(row.ID),
		Name:                row.Name,
		PlaybookID:          fromPgUUID(row.PlaybookID),
		PlaybookName:        name,
		PlaybookTriggerType: trigger,
		PlaybookDeleted:     row.PlaybookDeleted,
		Cron:                row.Cron,
		Timezone:            row.Timezone,
		Enabled:             row.Enabled,
		Parameters:          params,
		NextRunAt:           row.NextRunAt.Time,
		LastRunAt:           fromPgTimestamptzPtr(row.LastRunAt),
		CreatedBy:           fromPgUUIDPtr(row.CreatedBy),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
	}
}
