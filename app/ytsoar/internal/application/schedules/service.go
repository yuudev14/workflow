// Package schedules runs manual playbooks on a crontab timetable. A tick each minute calls
// RunDue; the schedules live in our table, River only gives the clock.
package schedules

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/ports_mock.go -package=mocks . ScheduleRepository,ScheduledRunner,PlaybookLookup

var ErrScheduleNotFound = apperr.New(apperr.NotFound, "schedule is not found")

const previewCount = 3

type ScheduleRepository interface {
	Create(ctx context.Context, s domain.Schedule) (domain.Schedule, error)
	// Get returns ErrScheduleNotFound when no row matches.
	Get(ctx context.Context, id uuid.UUID) (domain.Schedule, error)
	List(ctx context.Context, playbookID *uuid.UUID) ([]domain.Schedule, error)
	// Update saves the editable fields and next_run_at. It returns ErrScheduleNotFound when no row matches.
	Update(ctx context.Context, s domain.Schedule) (domain.Schedule, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// ListDueIDs skips the schedules of a deleted or a non-manual playbook.
	ListDueIDs(ctx context.Context, now time.Time) ([]uuid.UUID, error)
	// LockDue returns false when the schedule is no longer due or another tick holds it.
	LockDue(ctx context.Context, id uuid.UUID, now time.Time) (domain.Schedule, bool, error)
	SetNextRun(ctx context.Context, id uuid.UUID, lastRunAt *time.Time, next time.Time) error
}

// ScheduledRunner queues one run in the transaction in ctx. playbooks.PlaybookApplicationService implements it.
type ScheduledRunner interface {
	RunSchedule(ctx context.Context, playbookID, scheduleID uuid.UUID, parameters map[string]any) error
}

// PlaybookLookup is playbooks.PlaybookService.
type PlaybookLookup interface {
	GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error)
}

type Service struct {
	logger    logger.Logger
	repo      ScheduleRepository
	runner    ScheduledRunner
	playbooks PlaybookLookup
	tx        contracts.TxManager
	now       func() time.Time
}

func NewService(log logger.Logger, repo ScheduleRepository, runner ScheduledRunner, playbooks PlaybookLookup, tx contracts.TxManager) *Service {
	return &Service{logger: log, repo: repo, runner: runner, playbooks: playbooks, tx: tx, now: time.Now}
}

type CreatePayload struct {
	Name       string         `json:"name" binding:"required,max=200"`
	PlaybookID string         `json:"playbook_id" binding:"required,uuid"`
	Cron       string         `json:"cron" binding:"required"`
	Timezone   string         `json:"timezone"`
	Enabled    *bool          `json:"enabled"`
	Parameters map[string]any `json:"parameters"`
}

type UpdatePayload struct {
	Name       *string        `json:"name" binding:"omitempty,min=1,max=200"`
	Cron       *string        `json:"cron"`
	Timezone   *string        `json:"timezone"`
	Enabled    *bool          `json:"enabled"`
	Parameters map[string]any `json:"parameters"`
}

type PreviewPayload struct {
	Cron     string `json:"cron" binding:"required"`
	Timezone string `json:"timezone"`
}

func (s *Service) Create(ctx context.Context, p CreatePayload, actorID *uuid.UUID) (domain.Schedule, error) {
	playbookID, err := uuid.Parse(p.PlaybookID)
	if err != nil {
		return domain.Schedule{}, apperr.New(apperr.Invalid, "playbook_id must be a uuid")
	}
	playbook, err := s.playbooks.GetPlaybookById(ctx, p.PlaybookID)
	if err != nil {
		return domain.Schedule{}, err
	}
	if playbook.TriggerType == nil || *playbook.TriggerType != string(domain.TriggerTypeManual) {
		return domain.Schedule{}, apperr.New(apperr.Invalid, "a schedule can run only a playbook with a manual trigger")
	}
	sched := domain.Schedule{
		Name: p.Name, PlaybookID: playbookID, Cron: p.Cron, Timezone: orUTC(p.Timezone),
		Enabled: p.Enabled == nil || *p.Enabled, Parameters: orEmpty(p.Parameters), CreatedBy: actorID,
	}
	if sched.NextRunAt, err = s.nextRun(sched.Cron, sched.Timezone); err != nil {
		return domain.Schedule{}, err
	}
	return s.repo.Create(ctx, sched)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, p UpdatePayload) (domain.Schedule, error) {
	sched, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Schedule{}, err
	}
	wasEnabled, oldCron, oldTimezone := sched.Enabled, sched.Cron, sched.Timezone
	if p.Name != nil {
		sched.Name = *p.Name
	}
	if p.Cron != nil {
		sched.Cron = *p.Cron
	}
	if p.Timezone != nil {
		sched.Timezone = orUTC(*p.Timezone)
	}
	if p.Enabled != nil {
		sched.Enabled = *p.Enabled
	}
	if p.Parameters != nil {
		sched.Parameters = p.Parameters
	}
	// a changed timetable, or a schedule that turns on again, counts from now, so it never runs the missed times
	if sched.Cron != oldCron || sched.Timezone != oldTimezone || (sched.Enabled && !wasEnabled) {
		if sched.NextRunAt, err = s.nextRun(sched.Cron, sched.Timezone); err != nil {
			return domain.Schedule{}, err
		}
	}
	return s.repo.Update(ctx, sched)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Schedule, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, playbookID *uuid.UUID) ([]domain.Schedule, error) {
	return s.repo.List(ctx, playbookID)
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Preview returns the next times of a timetable, so the UI needs no cron library.
func (s *Service) Preview(_ context.Context, p PreviewPayload) ([]time.Time, error) {
	return domain.NextRuns(p.Cron, orUTC(p.Timezone), s.now(), previewCount)
}

// RunDue queues a run for each due schedule. Each schedule has its own transaction, so a
// bad one does not block the others. It does not run the times missed during a downtime.
func (s *Service) RunDue(ctx context.Context) error {
	now := s.now()
	ids, err := s.repo.ListDueIDs(ctx, now)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.runOne(ctx, id, now); err != nil {
			s.logger.Errorw("could not run a schedule, skipping this time", "schedule_id", id, "error", err)
			if err := s.skip(ctx, id, now); err != nil {
				s.logger.Errorw("could not move a schedule forward", "schedule_id", id, "error", err)
			}
		}
	}
	return nil
}

func (s *Service) runOne(ctx context.Context, id uuid.UUID, now time.Time) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		sched, ok, err := s.repo.LockDue(ctx, id, now)
		if err != nil || !ok {
			return err
		}
		next, err := s.nextRunFrom(sched.Cron, sched.Timezone, now)
		if err != nil {
			return err
		}
		params := make(map[string]any, len(sched.Parameters)+1)
		for k, v := range sched.Parameters {
			params[k] = v
		}
		params["scheduled_at"] = sched.NextRunAt.UTC().Format(time.RFC3339)
		if err := s.runner.RunSchedule(ctx, sched.PlaybookID, sched.ID, params); err != nil {
			return err
		}
		return s.repo.SetNextRun(ctx, id, &sched.NextRunAt, next)
	})
}

// skip moves next_run_at forward without a run
func (s *Service) skip(ctx context.Context, id uuid.UUID, now time.Time) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		sched, ok, err := s.repo.LockDue(ctx, id, now)
		if err != nil || !ok {
			return err
		}
		next, err := s.nextRunFrom(sched.Cron, sched.Timezone, now)
		if err != nil {
			return err
		}
		return s.repo.SetNextRun(ctx, id, nil, next)
	})
}

func (s *Service) nextRun(spec, timezone string) (time.Time, error) {
	return s.nextRunFrom(spec, timezone, s.now())
}

func (s *Service) nextRunFrom(spec, timezone string, after time.Time) (time.Time, error) {
	times, err := domain.NextRuns(spec, timezone, after, 1)
	if err != nil {
		return time.Time{}, err
	}
	if len(times) == 0 {
		return time.Time{}, apperr.New(apperr.Invalid, "cron never matches a date")
	}
	return times[0], nil
}

func orUTC(tz string) string {
	if tz == "" {
		return "UTC"
	}
	return tz
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
