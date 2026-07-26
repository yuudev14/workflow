package incidents_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	mock_contracts "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	mock_incidents "github.com/yuudev14/ytsoar/internal/application/incidents/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type publishedEvent struct {
	module string
	event  string
}

type testEnv struct {
	service   *incidents.Service
	repo      *mock_incidents.MockIncidentRepository
	timeline  *mock_incidents.MockAlertTimeline
	published *[]publishedEvent
	txCalls   *int
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	repo := mock_incidents.NewMockIncidentRepository(ctrl)
	timeline := mock_incidents.NewMockAlertTimeline(ctrl)

	published := []publishedEvent{}
	events := mock_contracts.NewMockModuleEventPublisher(ctrl)
	events.EXPECT().
		Publish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(module, event string, _ any) error {
			published = append(published, publishedEvent{module, event})
			return nil
		}).
		AnyTimes()

	txCalls := 0
	tx := mock_contracts.NewMockTxManager(ctrl)
	tx.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		}).
		AnyTimes()

	return &testEnv{
		service:   incidents.NewService(logger.NewNop(), repo, timeline, tx, events),
		repo:      repo,
		timeline:  timeline,
		published: &published,
		txCalls:   &txCalls,
	}
}

// The alerts service depends on incidents through a port it owns, so this is
// the compile-time proof the wiring in cmd/api/main.go stays valid.
var _ alerts.IncidentLinker = (*incidents.Service)(nil)

var _ contracts.ModuleEventPublisher = (*mock_contracts.MockModuleEventPublisher)(nil)

func TestCreateLinksSuppliedAlertsInOneTransaction(t *testing.T) {
	env := setup(t)
	incident := domain.Incident{ID: uuid.New(), Title: "Credential theft"}
	alertA, alertB := uuid.New(), uuid.New()

	env.repo.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.CreateParams) (domain.Incident, error) {
			assert.Equal(t, domain.IncidentStatusOpen, p.Status)
			assert.Equal(t, domain.SeverityHigh, p.Severity)
			return incident, nil
		})
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeCreated, p.Type)
			return nil
		})

	for _, id := range []uuid.UUID{alertA, alertB} {
		env.repo.EXPECT().LinkAlert(gomock.Any(), incident.ID, id, domain.LinkSourceManual).
			Return(true, nil)
	}
	// linked is written on both sides: the incident's timeline and the alert's.
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeLinked, p.Type)
			return nil
		}).Times(2)
	env.timeline.EXPECT().
		AppendAlertEvent(gomock.Any(), gomock.Any(), domain.EventTypeLinked, gomock.Any(), gomock.Any()).
		Return(nil).Times(2)

	got, err := env.service.Create(context.Background(), incidents.CreateIncidentPayload{
		Title:    "Credential theft",
		Severity: "high",
		AlertIDs: []string{alertA.String(), alertB.String()},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, incident.ID, got.ID)
	assert.Equal(t, 1, *env.txCalls)
	assert.Equal(t, []publishedEvent{{domain.ModuleEventIncident, domain.ModuleEventCreated}}, *env.published)
}

func TestCreateRejectsMalformedAlertIDsBeforeOpeningATransaction(t *testing.T) {
	env := setup(t)

	_, err := env.service.Create(context.Background(), incidents.CreateIncidentPayload{
		Title:    "Credential theft",
		Severity: "high",
		AlertIDs: []string{"not-a-uuid"},
	}, nil)

	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
	assert.Equal(t, 0, *env.txCalls)
}

func TestCreateDoesNotPublishWhenLinkFails(t *testing.T) {
	env := setup(t)
	boom := errors.New("link failed")
	incident := domain.Incident{ID: uuid.New()}

	env.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(incident, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(nil)
	env.repo.EXPECT().LinkAlert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, boom)

	_, err := env.service.Create(context.Background(), incidents.CreateIncidentPayload{
		Title:    "Credential theft",
		Severity: "high",
		AlertIDs: []string{uuid.NewString()},
	}, nil)

	require.ErrorIs(t, err, boom)
	assert.Empty(t, *env.published)
}

// The stepper lives in the domain so a bad move is a 400, not a silent write.
func TestUpdateStatusEnforcesTheTransitionStepper(t *testing.T) {
	env := setup(t)
	id := uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Incident{ID: id, Status: domain.IncidentStatusContained}, nil)
	env.repo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Times(0)

	_, err := env.service.UpdateStatus(context.Background(), id,
		incidents.UpdateIncidentStatusPayload{Status: "open"}, nil)

	require.Error(t, err)
	kind, msg := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
	assert.Contains(t, msg, "contained")
	assert.Empty(t, *env.published, "a rejected transition must not announce a change")
}

func TestUpdateStatusAllowsForwardSkipAndReopen(t *testing.T) {
	cases := []struct {
		from domain.IncidentStatus
		to   string
	}{
		{domain.IncidentStatusOpen, "contained"},         // analysts contain before recording investigation
		{domain.IncidentStatusResolved, "investigating"}, // reopened
		{domain.IncidentStatusClosed, "investigating"},
		{domain.IncidentStatusOpen, "open"}, // idempotent retry
	}

	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+tc.to, func(t *testing.T) {
			env := setup(t)
			id := uuid.New()

			env.repo.EXPECT().GetByID(gomock.Any(), id).
				Return(domain.Incident{ID: id, Status: tc.from}, nil)
			env.repo.EXPECT().UpdateStatus(gomock.Any(), id, domain.IncidentStatus(tc.to)).
				Return(domain.Incident{ID: id, Status: domain.IncidentStatus(tc.to)}, nil)
			env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
					var body map[string]any
					require.NoError(t, json.Unmarshal(p.Body, &body))
					assert.Equal(t, string(tc.from), body["from"])
					assert.Equal(t, tc.to, body["to"])
					return nil
				})

			_, err := env.service.UpdateStatus(context.Background(), id,
				incidents.UpdateIncidentStatusPayload{Status: tc.to}, nil)
			require.NoError(t, err)
			assert.Equal(t, []publishedEvent{{domain.ModuleEventIncident, domain.ModuleEventUpdated}}, *env.published)
		})
	}
}

func TestUpdateStatusRejectsUnknownStatus(t *testing.T) {
	env := setup(t)

	_, err := env.service.UpdateStatus(context.Background(), uuid.New(),
		incidents.UpdateIncidentStatusPayload{Status: "banana"}, nil)

	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
	assert.Equal(t, 0, *env.txCalls)
}

// Escalation runs inside the alert service's transaction, so it must not open
// one of its own or publish — the caller announces both entities after commit.
func TestCreateForEscalationDoesNotOpenATransactionOrPublish(t *testing.T) {
	env := setup(t)
	incident := domain.Incident{ID: uuid.New(), Title: "Encoded PowerShell execution"}
	alertID := uuid.New()

	env.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(incident, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeCreated, p.Type)
			return nil
		})
	env.repo.EXPECT().LinkAlert(gomock.Any(), incident.ID, alertID, domain.LinkSourceEscalate).
		Return(true, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeLinked, p.Type)
			return nil
		})

	got, err := env.service.CreateForEscalation(context.Background(), alerts.EscalationParams{
		Title:    "Encoded PowerShell execution",
		Severity: domain.SeverityHigh,
		AlertID:  alertID,
	})

	require.NoError(t, err)
	assert.Equal(t, incident.ID, got.ID)
	assert.Equal(t, 0, *env.txCalls, "it must join the caller's transaction, not start one")
	assert.Empty(t, *env.published, "the caller publishes once the whole escalation commits")
}

func TestLinkAlertWritesBothTimelines(t *testing.T) {
	env := setup(t)
	incidentID, alertID := uuid.New(), uuid.New()
	incident := domain.Incident{ID: incidentID, Title: "Credential theft"}

	env.repo.EXPECT().GetByID(gomock.Any(), incidentID).Return(incident, nil)
	env.repo.EXPECT().LinkAlert(gomock.Any(), incidentID, alertID, domain.LinkSourceManual).
		Return(true, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeLinked, p.Type)
			var body map[string]any
			require.NoError(t, json.Unmarshal(p.Body, &body))
			assert.Equal(t, alertID.String(), body["alert_id"])
			return nil
		})
	env.timeline.EXPECT().
		AppendAlertEvent(gomock.Any(), alertID, domain.EventTypeLinked, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ domain.EventType, _ *uuid.UUID, body []byte) error {
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(body, &decoded))
			assert.Equal(t, incidentID.String(), decoded["incident_id"])
			assert.Equal(t, "Credential theft", decoded["incident_title"])
			return nil
		})

	err := env.service.LinkAlert(context.Background(), incidentID,
		incidents.LinkAlertPayload{AlertID: alertID.String()}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, *env.txCalls)
}

// ON CONFLICT DO NOTHING makes a repeat link a no-op, but it must not append a
// second pair of timeline rows claiming it happened twice.
func TestLinkAlertIsIdempotent(t *testing.T) {
	env := setup(t)
	incidentID, alertID := uuid.New(), uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), incidentID).
		Return(domain.Incident{ID: incidentID}, nil)
	env.repo.EXPECT().LinkAlert(gomock.Any(), incidentID, alertID, domain.LinkSourceManual).
		Return(false, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Times(0)
	env.timeline.EXPECT().AppendAlertEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	err := env.service.LinkAlert(context.Background(), incidentID,
		incidents.LinkAlertPayload{AlertID: alertID.String()}, nil)
	require.NoError(t, err)
}

func TestUnlinkAlertWritesBothTimelines(t *testing.T) {
	env := setup(t)
	incidentID, alertID := uuid.New(), uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), incidentID).
		Return(domain.Incident{ID: incidentID, Title: "Credential theft"}, nil)
	env.repo.EXPECT().UnlinkAlert(gomock.Any(), incidentID, alertID).Return(nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p incidents.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeUnlinked, p.Type)
			return nil
		})
	env.timeline.EXPECT().
		AppendAlertEvent(gomock.Any(), alertID, domain.EventTypeUnlinked, gomock.Any(), gomock.Any()).
		Return(nil)

	require.NoError(t, env.service.UnlinkAlert(context.Background(), incidentID, alertID, nil))
	assert.Equal(t, []publishedEvent{{domain.ModuleEventIncident, domain.ModuleEventUpdated}}, *env.published)
}

func TestUnlinkAlertSurfacesNotLinked(t *testing.T) {
	env := setup(t)
	incidentID, alertID := uuid.New(), uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), incidentID).Return(domain.Incident{ID: incidentID}, nil)
	env.repo.EXPECT().UnlinkAlert(gomock.Any(), incidentID, alertID).
		Return(incidents.ErrAlertNotLinked)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Times(0)

	err := env.service.UnlinkAlert(context.Background(), incidentID, alertID, nil)
	require.ErrorIs(t, err, incidents.ErrAlertNotLinked)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.NotFound, kind)
	assert.Empty(t, *env.published)
}

func TestUpdateNoteRejectsNonAuthors(t *testing.T) {
	env := setup(t)
	noteID := uuid.New()
	author, other := uuid.New(), uuid.New()

	env.repo.EXPECT().GetNote(gomock.Any(), noteID).
		Return(domain.IncidentNote{ID: noteID, AuthorID: &author}, nil)
	env.repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := env.service.UpdateNote(context.Background(), noteID,
		incidents.UpdateNotePayload{Body: "rewritten"}, &other)

	require.ErrorIs(t, err, incidents.ErrNotNoteAuthor)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Forbidden, kind)
}

func TestAddNoteRejectsBlankBody(t *testing.T) {
	env := setup(t)

	_, err := env.service.AddNote(context.Background(), uuid.New(),
		incidents.AddNotePayload{Body: "  "}, nil)

	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
}

func TestListClampsLimitAndReportsTotal(t *testing.T) {
	env := setup(t)
	next := "opaque-token"

	env.repo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f incidents.IncidentFilter) ([]incidents.IncidentListItem, *string, error) {
			assert.Equal(t, incidents.MaxLimit, f.Limit)
			return []incidents.IncidentListItem{{ID: uuid.New()}}, &next, nil
		})
	env.repo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(7, nil)

	page, err := env.service.List(context.Background(), incidents.IncidentFilter{Limit: 9999})
	require.NoError(t, err)
	assert.Equal(t, 7, page.Total)
	assert.Equal(t, &next, page.NextCursor)
}
