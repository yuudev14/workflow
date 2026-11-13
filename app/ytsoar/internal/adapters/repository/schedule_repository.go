package repository

import (
	"context"
	"encoding/json"
	"time"

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
		Name: s.Name, PlaybookID: toPgUUID(s.PlaybookID), Cron: s.Cron, Timezone: s.Timezone,
		Enabled: s.Enabled, Parameters: params, NextRunAt: toPgTimestamptz(s.NextRunAt),
		CreatedBy: toPgUUIDPtr(s.CreatedBy),
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

func (r *ScheduleRepositoryImpl) List(ctx context.Context, playbookID *uuid.UUID) ([]domain.Schedule, error) {
	rows, err := r.queriesFromContext(ctx).ListSchedules(ctx, toPgUUIDPtr(playbookID))
	if err != nil {
		return nil, err
	}
	list := make([]domain.Schedule, len(rows))
	for i, row := range rows {
		list[i] = toDomainSchedule(db.GetScheduleRow(row))
	}
	return list, nil
}

func (r *ScheduleRepositoryImpl) Update(ctx context.Context, s domain.Schedule) (domain.Schedule, error) {
	params, err := json.Marshal(s.Parameters)
	if err != nil {
		return domain.Schedule{}, err
	}
	id, err := r.queriesFromContext(ctx).UpdateSchedule(ctx, db.UpdateScheduleParams{
		ID: toPgUUID(s.ID), Name: s.Name, Cron: s.Cron, Timezone: s.Timezone, Enabled: s.Enabled,
		Parameters: params, NextRunAt: toPgTimestamptz(s.NextRunAt),
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
		ID: row.ID, Name: row.Name, PlaybookID: row.PlaybookID, Cron: row.Cron, Timezone: row.Timezone,
		Enabled: row.Enabled, Parameters: row.Parameters, NextRunAt: row.NextRunAt, LastRunAt: row.LastRunAt,
		CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
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
		ID: fromPgUUID(row.ID), Name: row.Name, PlaybookID: fromPgUUID(row.PlaybookID),
		PlaybookName: name, PlaybookTriggerType: trigger, PlaybookDeleted: row.PlaybookDeleted,
		Cron: row.Cron, Timezone: row.Timezone, Enabled: row.Enabled, Parameters: params,
		NextRunAt: row.NextRunAt.Time, LastRunAt: fromPgTimestamptzPtr(row.LastRunAt),
		CreatedBy: fromPgUUIDPtr(row.CreatedBy), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}
