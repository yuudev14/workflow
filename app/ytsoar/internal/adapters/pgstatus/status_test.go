package pgstatus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type fakeReaders struct{}

func (fakeReaders) GetPlaybookHistoryById(_ context.Context, id uuid.UUID) (*domain.PlaybookHistoryResponse, error) {
	return &domain.PlaybookHistoryResponse{ID: id}, nil
}

func (fakeReaders) GetTaskHistoryById(_ context.Context, id uuid.UUID) (*domain.TaskHistory, error) {
	return &domain.TaskHistory{ID: id}, nil
}

type fakeHub chan any

func (h fakeHub) Broadcast(data any) { h <- data }

// This test needs a Postgres: TEST_DATABASE_URL=postgres://... go test ./internal/adapters/pgstatus
func TestNotifierReachesTheListenersHub(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := db.NewPool(ctx, url)
	require.NoError(t, err)
	defer pool.Close()

	hub := make(fakeHub, 2)
	go NewListener(logger.NewNop(), pool, fakeReaders{}, fakeReaders{}, hub).Start(ctx)
	time.Sleep(500 * time.Millisecond) // let it LISTEN before the first NOTIFY

	taskID, runID := uuid.New(), uuid.New()
	require.NoError(t, NewNotifier(pool).Publish(ctx, "task_status", taskID))
	require.NoError(t, NewNotifier(pool).Publish(ctx, "playbook_status", runID))

	for _, want := range []struct {
		event string
		id    uuid.UUID
	}{{"task_status", taskID}, {"playbook_status", runID}} {
		select {
		case got := <-hub:
			envelope := got.(map[string]any)
			assert.Equal(t, want.event, envelope["event"])
			switch row := envelope["data"].(type) {
			case *domain.TaskHistory:
				assert.Equal(t, want.id, row.ID)
			case *domain.PlaybookHistoryResponse:
				assert.Equal(t, want.id, row.ID)
			default:
				t.Fatalf("unexpected data %T", row)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no %s event reached the hub", want.event)
		}
	}
}
