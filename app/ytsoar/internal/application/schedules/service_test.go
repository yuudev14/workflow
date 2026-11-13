package schedules_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/schedules"
	"github.com/yuudev14/ytsoar/internal/application/schedules/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type fakeTx struct{}

func (fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixture struct {
	svc       *schedules.Service
	repo      *mocks.MockScheduleRepository
	runner    *mocks.MockScheduledRunner
	playbooks *mocks.MockPlaybookLookup
}

func newFixture(t *testing.T) fixture {
	ctrl := gomock.NewController(t)
	f := fixture{
		repo:      mocks.NewMockScheduleRepository(ctrl),
		runner:    mocks.NewMockScheduledRunner(ctrl),
		playbooks: mocks.NewMockPlaybookLookup(ctrl),
	}
	f.svc = schedules.NewService(logger.NewNop(), f.repo, f.runner, f.playbooks, fakeTx{})
	return f
}

func trigger(s string) *string { return &s }

func TestCreate(t *testing.T) {
	playbookID := uuid.New()
	payload := schedules.CreatePayload{Name: "nightly", PlaybookID: playbookID.String(), Cron: "0 2 * * *"}

	t.Run("saves the schedule with the next run time", func(t *testing.T) {
		f := newFixture(t)
		f.playbooks.EXPECT().GetPlaybookById(gomock.Any(), playbookID.String()).Return(&domain.Playbooks{TriggerType: trigger("manual")}, nil)
		f.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s domain.Schedule) (domain.Schedule, error) {
			assert.Equal(t, "UTC", s.Timezone)
			assert.True(t, s.Enabled)
			assert.True(t, s.NextRunAt.After(time.Now()))
			return s, nil
		})
		_, err := f.svc.Create(context.Background(), payload, nil)
		require.NoError(t, err)
	})

	t.Run("refuses a playbook that is not manual", func(t *testing.T) {
		f := newFixture(t)
		f.playbooks.EXPECT().GetPlaybookById(gomock.Any(), gomock.Any()).Return(&domain.Playbooks{TriggerType: trigger("webhook")}, nil)
		_, err := f.svc.Create(context.Background(), payload, nil)
		kind, _ := apperr.KindOf(err)
		assert.Equal(t, apperr.Invalid, kind)
	})

	t.Run("refuses a bad cron", func(t *testing.T) {
		f := newFixture(t)
		f.playbooks.EXPECT().GetPlaybookById(gomock.Any(), gomock.Any()).Return(&domain.Playbooks{TriggerType: trigger("manual")}, nil)
		bad := payload
		bad.Cron = "@daily"
		_, err := f.svc.Create(context.Background(), bad, nil)
		assert.ErrorIs(t, err, domain.ErrInvalidCron)
	})
}

func TestUpdate(t *testing.T) {
	id := uuid.New()
	old := time.Now().Add(-time.Hour)
	stored := domain.Schedule{ID: id, Cron: "0 2 * * *", Timezone: "UTC", Enabled: true, NextRunAt: old}

	t.Run("keeps next_run_at when the timetable does not change", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().Get(gomock.Any(), id).Return(stored, nil)
		f.repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s domain.Schedule) (domain.Schedule, error) {
			assert.Equal(t, old, s.NextRunAt)
			return s, nil
		})
		name := "renamed"
		_, err := f.svc.Update(context.Background(), id, schedules.UpdatePayload{Name: &name})
		require.NoError(t, err)
	})

	t.Run("counts the next run from now when the schedule turns on again", func(t *testing.T) {
		f := newFixture(t)
		disabled := stored
		disabled.Enabled = false
		f.repo.EXPECT().Get(gomock.Any(), id).Return(disabled, nil)
		f.repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s domain.Schedule) (domain.Schedule, error) {
			assert.True(t, s.NextRunAt.After(time.Now()))
			return s, nil
		})
		on := true
		_, err := f.svc.Update(context.Background(), id, schedules.UpdatePayload{Enabled: &on})
		require.NoError(t, err)
	})
}

func TestRunDue(t *testing.T) {
	id, playbookID := uuid.New(), uuid.New()
	due := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	sched := domain.Schedule{ID: id, PlaybookID: playbookID, Cron: "0 2 * * *", Timezone: "UTC", Enabled: true,
		Parameters: map[string]any{"env": "prod"}, NextRunAt: due}

	t.Run("queues a run with scheduled_at and moves next_run_at forward", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().ListDueIDs(gomock.Any(), gomock.Any()).Return([]uuid.UUID{id}, nil)
		f.repo.EXPECT().LockDue(gomock.Any(), id, gomock.Any()).Return(sched, true, nil)
		f.runner.EXPECT().RunSchedule(gomock.Any(), playbookID, id, map[string]any{
			"env": "prod", "scheduled_at": due.Format(time.RFC3339),
		}).Return(nil)
		f.repo.EXPECT().SetNextRun(gomock.Any(), id, &due, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, _ *time.Time, next time.Time) error {
				assert.True(t, next.After(time.Now()))
				return nil
			})
		require.NoError(t, f.svc.RunDue(context.Background()))
	})

	t.Run("moves a failing schedule forward without a run and goes on with the others", func(t *testing.T) {
		f := newFixture(t)
		other := uuid.New()
		f.repo.EXPECT().ListDueIDs(gomock.Any(), gomock.Any()).Return([]uuid.UUID{id, other}, nil)
		f.repo.EXPECT().LockDue(gomock.Any(), id, gomock.Any()).Return(sched, true, nil).Times(2)
		f.runner.EXPECT().RunSchedule(gomock.Any(), playbookID, id, gomock.Any()).Return(errors.New("no start step"))
		f.repo.EXPECT().SetNextRun(gomock.Any(), id, nil, gomock.Any()).Return(nil)

		otherSched := sched
		otherSched.ID = other
		f.repo.EXPECT().LockDue(gomock.Any(), other, gomock.Any()).Return(otherSched, true, nil)
		f.runner.EXPECT().RunSchedule(gomock.Any(), playbookID, other, gomock.Any()).Return(nil)
		f.repo.EXPECT().SetNextRun(gomock.Any(), other, gomock.Any(), gomock.Any()).Return(nil)
		require.NoError(t, f.svc.RunDue(context.Background()))
	})

	t.Run("does nothing when another tick already took the schedule", func(t *testing.T) {
		f := newFixture(t)
		f.repo.EXPECT().ListDueIDs(gomock.Any(), gomock.Any()).Return([]uuid.UUID{id}, nil)
		f.repo.EXPECT().LockDue(gomock.Any(), id, gomock.Any()).Return(domain.Schedule{}, false, nil)
		require.NoError(t, f.svc.RunDue(context.Background()))
	})
}
