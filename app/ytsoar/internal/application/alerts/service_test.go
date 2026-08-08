package alerts_test

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
	mock_alerts "github.com/yuudev14/ytsoar/internal/application/alerts/mocks"
	mock_contracts "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type publishedEvent struct {
	module string
	event  string
}

type testEnv struct {
	service   *alerts.Service
	repo      *mock_alerts.MockAlertRepository
	incidents *mock_alerts.MockIncidentLinker
	published *[]publishedEvent
	txCalls   *int
	// txCommits counts transactions whose closure returned nil, so a test can
	// prove a publish only followed a successful commit.
	txCommits *int
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	repo := mock_alerts.NewMockAlertRepository(ctrl)
	incidents := mock_alerts.NewMockIncidentLinker(ctrl)

	published := []publishedEvent{}
	events := mock_contracts.NewMockModuleEventPublisher(ctrl)
	events.EXPECT().
		Publish(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(module, event string, _ any) error {
			published = append(published, publishedEvent{module, event})
			return nil
		}).
		AnyTimes()

	txCalls, txCommits := 0, 0
	tx := mock_contracts.NewMockTxManager(ctrl)
	tx.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			err := fn(ctx)
			if err == nil {
				txCommits++
			}
			return err
		}).
		AnyTimes()

	// The directory only labels assignee changes; returning nothing keeps the
	// diff assertions about the diff.
	users := mock_contracts.NewMockUserDirectory(ctrl)
	users.EXPECT().
		UsernamesByIDs(gomock.Any(), gomock.Any()).
		Return(map[uuid.UUID]string{}, nil).
		AnyTimes()

	return &testEnv{
		service:   alerts.NewService(logger.NewNop(), repo, incidents, tx, events, users),
		repo:      repo,
		incidents: incidents,
		published: &published,
		txCalls:   &txCalls,
		txCommits: &txCommits,
	}
}

func createPayload() alerts.CreateAlertPayload {
	return alerts.CreateAlertPayload{
		Title:      "Encoded PowerShell execution",
		Severity:   "high",
		SourceKind: "edr",
		Reporter:   new("CrowdStrike"),
		Payload:    json.RawMessage(`{"host":{"hostname":"WIN-DC01"}}`),
	}
}

func TestFingerprintIsStableAndEntityScoped(t *testing.T) {
	kind := domain.SourceKindEDR
	reporter := new("CrowdStrike")
	title := "Encoded PowerShell execution"

	hostA := json.RawMessage(`{"host":{"hostname":"WIN-DC01"},"@timestamp":"2026-07-25T10:00:00Z"}`)
	hostASameAlertLater := json.RawMessage(`{"host":{"hostname":"WIN-DC01"},"@timestamp":"2026-07-25T11:30:00Z"}`)
	hostB := json.RawMessage(`{"host":{"hostname":"WIN-WS42"}}`)

	base := alerts.Fingerprint(kind, reporter, title, hostA)

	assert.Equal(t, base, alerts.Fingerprint(kind, reporter, title, hostASameAlertLater),
		"timestamps must not enter the fingerprint or nothing would ever dedup")

	// Without the primary entity, one rule firing across 200 hosts would
	// collapse into a single alert.
	assert.NotEqual(t, base, alerts.Fingerprint(kind, reporter, title, hostB))
	assert.NotEqual(t, base, alerts.Fingerprint(kind, reporter, "Different rule", hostA))
	assert.NotEqual(t, base, alerts.Fingerprint(domain.SourceKindFirewall, reporter, title, hostA))
	assert.NotEqual(t, base, alerts.Fingerprint(kind, new("SentinelOne"), title, hostA))
}

// A fresh row is alert.created; a dedup hit is alert.updated. Getting this
// wrong makes a 10k-alert storm fire every on_create playbook 10k times.
func TestCreateInsertPublishesCreatedAndWritesTimeline(t *testing.T) {
	env := setup(t)
	alert := domain.Alert{ID: uuid.New(), Title: "Encoded PowerShell execution"}

	env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(alert, true, nil)
	env.repo.EXPECT().
		AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeCreated, p.Type)
			assert.Equal(t, alert.ID, p.AlertID)
			return nil
		})

	got, err := env.service.Create(context.Background(), createPayload(), nil)
	require.NoError(t, err)
	assert.Equal(t, alert.ID, got.ID)
	assert.Equal(t, []publishedEvent{{domain.ModuleEventAlert, domain.ModuleEventCreated}}, *env.published)
}

func TestCreateDedupPublishesUpdatedAndWritesNoTimelineRow(t *testing.T) {
	env := setup(t)
	alert := domain.Alert{ID: uuid.New(), DedupCount: 2}

	env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(alert, false, nil)
	// A recurrence is not a timeline entry - dedup_count/last_seen carry it.
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Times(0)

	_, err := env.service.Create(context.Background(), createPayload(), nil)
	require.NoError(t, err)
	assert.Equal(t, []publishedEvent{{domain.ModuleEventAlert, domain.ModuleEventUpdated}}, *env.published)
}

// Publishing inside the transaction would announce state a rollback discards.
func TestCreateDoesNotPublishWhenTransactionFails(t *testing.T) {
	env := setup(t)
	boom := errors.New("append failed")

	env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(domain.Alert{ID: uuid.New()}, true, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(boom)

	_, err := env.service.Create(context.Background(), createPayload(), nil)
	require.ErrorIs(t, err, boom)
	assert.Empty(t, *env.published, "no event may escape a rolled-back transaction")
	assert.Equal(t, 0, *env.txCommits)
}

// One bad item must not roll back a whole forwarder batch.
func TestCreateBatchIsolatesFailures(t *testing.T) {
	env := setup(t)
	ok := domain.Alert{ID: uuid.New()}

	gomock.InOrder(
		env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(ok, true, nil),
		env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(domain.Alert{}, false, errors.New("bad row")),
		env.repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(ok, true, nil),
	)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	created, failed, err := env.service.CreateBatch(context.Background(),
		alerts.BatchCreateAlertPayload{Alerts: []alerts.CreateAlertPayload{
			createPayload(), createPayload(), createPayload(),
		}}, nil)

	require.NoError(t, err)
	assert.Len(t, created, 2)
	assert.Equal(t, 1, failed)
	assert.Len(t, *env.published, 2)
}

func TestUpdateWritesFieldDiffToTimeline(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	actor := uuid.New()
	assignee := uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Alert{ID: id, Severity: domain.SeverityLow}, nil)
	env.repo.EXPECT().Update(gomock.Any(), id, gomock.Any()).
		Return(domain.Alert{
			ID: id, Severity: domain.SeverityCritical, AssigneeID: &assignee,
		}, nil)
	env.repo.EXPECT().
		AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeUpdated, p.Type)
			assert.Equal(t, &actor, p.ActorID)

			var body struct {
				Changes []domain.FieldChange `json:"changes"`
			}
			require.NoError(t, json.Unmarshal(p.Body, &body))
			require.Len(t, body.Changes, 2)

			assert.Equal(t, "severity", body.Changes[0].Field)
			assert.Equal(t, "low", body.Changes[0].From)
			assert.Equal(t, "critical", body.Changes[0].To)

			assert.Equal(t, "assignee_id", body.Changes[1].Field)
			assert.Nil(t, body.Changes[1].From)
			assert.Equal(t, assignee.String(), body.Changes[1].To)
			return nil
		})

	sev := "critical"
	_, err := env.service.Update(context.Background(), id, alerts.UpdateAlertPayload{
		Severity: types.Nullable[string]{Value: &sev, Set: true},
	}, &actor)
	require.NoError(t, err)
}

// A PATCH that changes nothing must not append an event: the timeline is an audit
// log, and a row saying "nothing happened" is noise a reviewer has to read past.
func TestUpdateWithNoActualChangeWritesNoEvent(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	unchanged := domain.Alert{ID: id, Severity: domain.SeverityHigh}

	env.repo.EXPECT().GetByID(gomock.Any(), id).Return(unchanged, nil)
	env.repo.EXPECT().Update(gomock.Any(), id, gomock.Any()).Return(unchanged, nil)
	// no AppendEvent expectation: gomock fails the test if one is called

	sev := "high"
	_, err := env.service.Update(context.Background(), id, alerts.UpdateAlertPayload{
		Severity: types.Nullable[string]{Value: &sev, Set: true},
	}, nil)
	require.NoError(t, err)
}

func TestUpdateStatusWritesTransitionToTimeline(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	actor := uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusNew}, nil)
	env.repo.EXPECT().UpdateStatus(gomock.Any(), id, domain.AlertStatusInvestigating, nil).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusInvestigating}, nil)
	env.repo.EXPECT().
		AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeStatusChanged, p.Type)
			assert.Equal(t, &actor, p.ActorID)

			var body map[string]any
			require.NoError(t, json.Unmarshal(p.Body, &body))
			assert.Equal(t, "new", body["from"])
			assert.Equal(t, "investigating", body["to"])
			assert.NotContains(t, body, "closure_note")
			return nil
		})

	_, err := env.service.UpdateStatus(context.Background(), id,
		alerts.UpdateAlertStatusPayload{Status: "investigating"}, &actor)
	require.NoError(t, err)
	assert.Equal(t, []publishedEvent{{domain.ModuleEventAlert, domain.ModuleEventUpdated}}, *env.published)
}

// The column is cleared on reopen, so the timeline event is the only durable
// record of why an alert was closed.
func TestUpdateStatusCarriesClosureNoteIntoTheTimeline(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	note := "Known admin tool, matches Tuesday rollout."

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusInvestigating}, nil)
	env.repo.EXPECT().
		UpdateStatus(gomock.Any(), id, domain.AlertStatusFalsePositive, &note).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusFalsePositive}, nil)
	env.repo.EXPECT().
		AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.AppendEventParams) error {
			var body map[string]any
			require.NoError(t, json.Unmarshal(p.Body, &body))
			assert.Equal(t, note, body["closure_note"])
			return nil
		})

	_, err := env.service.UpdateStatus(context.Background(), id,
		alerts.UpdateAlertStatusPayload{Status: "falsepos", ClosureNote: &note}, nil)
	require.NoError(t, err)
}

func TestUpdateStatusTreatsBlankClosureNoteAsAbsent(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	blank := "   "

	env.repo.EXPECT().GetByID(gomock.Any(), id).Return(domain.Alert{ID: id}, nil)
	env.repo.EXPECT().UpdateStatus(gomock.Any(), id, domain.AlertStatusClosed, nil).
		Return(domain.Alert{ID: id}, nil)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(nil)

	_, err := env.service.UpdateStatus(context.Background(), id,
		alerts.UpdateAlertStatusPayload{Status: "closed", ClosureNote: &blank}, nil)
	require.NoError(t, err)
}

func TestUpdateStatusRejectsUnknownStatusBeforeTouchingTheRepo(t *testing.T) {
	env := setup(t)

	_, err := env.service.UpdateStatus(context.Background(), uuid.New(),
		alerts.UpdateAlertStatusPayload{Status: "banana"}, nil)

	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
	assert.Equal(t, 0, *env.txCalls)
}

// Escalation is one transaction: incident, link, alert status and both timeline
// rows either all land or none do.
func TestEscalateRunsInOneTransactionAndPublishesBothEntities(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	actor := uuid.New()
	team := uuid.New()
	incident := domain.Incident{ID: uuid.New(), Title: "Encoded PowerShell execution"}

	env.repo.EXPECT().GetByID(gomock.Any(), id).Return(domain.Alert{
		ID:       id,
		Title:    "Encoded PowerShell execution",
		Severity: domain.SeverityHigh,
		Status:   domain.AlertStatusNew,
		TeamID:   &team,
	}, nil)

	env.incidents.EXPECT().
		CreateForEscalation(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.EscalationParams) (domain.Incident, error) {
			assert.Equal(t, "Encoded PowerShell execution", p.Title)
			assert.Equal(t, domain.SeverityHigh, p.Severity)
			assert.Equal(t, id, p.AlertID)
			assert.Equal(t, &team, p.TeamID)
			return incident, nil
		})

	// A new alert is moved out of the queue by escalating it.
	env.repo.EXPECT().UpdateStatus(gomock.Any(), id, domain.AlertStatusInvestigating, nil).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusInvestigating}, nil)
	env.repo.EXPECT().
		AppendEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.AppendEventParams) error {
			assert.Equal(t, domain.EventTypeEscalated, p.Type)
			var body map[string]any
			require.NoError(t, json.Unmarshal(p.Body, &body))
			assert.Equal(t, incident.ID.String(), body["incident_id"])
			return nil
		})

	got, err := env.service.Escalate(context.Background(), id, alerts.EscalateAlertPayload{}, &actor)
	require.NoError(t, err)
	assert.Equal(t, incident.ID, got.ID)
	assert.Equal(t, 1, *env.txCalls, "escalation must be a single transaction")
	assert.Equal(t, []publishedEvent{
		{domain.ModuleEventIncident, domain.ModuleEventCreated},
		{domain.ModuleEventAlert, domain.ModuleEventUpdated},
	}, *env.published)
}

func TestEscalateKeepsANonNewStatus(t *testing.T) {
	env := setup(t)
	id := uuid.New()

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Alert{ID: id, Status: domain.AlertStatusInvestigating}, nil)
	env.incidents.EXPECT().CreateForEscalation(gomock.Any(), gomock.Any()).
		Return(domain.Incident{ID: uuid.New()}, nil)
	env.repo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(nil)

	_, err := env.service.Escalate(context.Background(), id, alerts.EscalateAlertPayload{}, nil)
	require.NoError(t, err)
}

func TestEscalateOverridesTitleWhenSupplied(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	title := "  Suspected credential theft on WIN-DC01  "

	env.repo.EXPECT().GetByID(gomock.Any(), id).
		Return(domain.Alert{ID: id, Title: "Encoded PowerShell execution", Status: domain.AlertStatusClosed}, nil)
	env.incidents.EXPECT().
		CreateForEscalation(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p alerts.EscalationParams) (domain.Incident, error) {
			assert.Equal(t, "Suspected credential theft on WIN-DC01", p.Title)
			return domain.Incident{ID: uuid.New()}, nil
		})
	env.repo.EXPECT().AppendEvent(gomock.Any(), gomock.Any()).Return(nil)

	_, err := env.service.Escalate(context.Background(), id,
		alerts.EscalateAlertPayload{Title: &title}, nil)
	require.NoError(t, err)
}

func TestEscalateDoesNotPublishWhenIncidentCreationFails(t *testing.T) {
	env := setup(t)
	id := uuid.New()
	boom := errors.New("incident insert failed")

	env.repo.EXPECT().GetByID(gomock.Any(), id).Return(domain.Alert{ID: id}, nil)
	env.incidents.EXPECT().CreateForEscalation(gomock.Any(), gomock.Any()).
		Return(domain.Incident{}, boom)

	_, err := env.service.Escalate(context.Background(), id, alerts.EscalateAlertPayload{}, nil)
	require.ErrorIs(t, err, boom)
	assert.Empty(t, *env.published)
}

func TestAddNoteRejectsBlankBody(t *testing.T) {
	env := setup(t)

	_, err := env.service.AddNote(context.Background(), uuid.New(),
		alerts.AddNotePayload{Body: "   \n "}, nil)

	require.Error(t, err)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Invalid, kind)
}

// A note is evidence in a post-incident review, so a third party must not be
// able to rewrite it.
func TestUpdateNoteRejectsNonAuthors(t *testing.T) {
	env := setup(t)
	noteID := uuid.New()
	author := uuid.New()
	other := uuid.New()

	env.repo.EXPECT().GetNote(gomock.Any(), noteID).
		Return(domain.AlertNote{ID: noteID, AuthorID: &author}, nil)
	env.repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := env.service.UpdateNote(context.Background(), noteID,
		alerts.UpdateNotePayload{Body: "rewritten"}, &other)

	require.ErrorIs(t, err, alerts.ErrNotNoteAuthor)
	kind, _ := apperr.KindOf(err)
	assert.Equal(t, apperr.Forbidden, kind)
}

// ON DELETE SET NULL on author_id means a deleted user leaves an ownerless
// note. It stays readable, but nobody inherits the right to edit it.
func TestUpdateNoteRejectsWhenAuthorWasDeleted(t *testing.T) {
	env := setup(t)
	noteID := uuid.New()
	actor := uuid.New()

	env.repo.EXPECT().GetNote(gomock.Any(), noteID).
		Return(domain.AlertNote{ID: noteID, AuthorID: nil}, nil)
	env.repo.EXPECT().UpdateNote(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := env.service.UpdateNote(context.Background(), noteID,
		alerts.UpdateNotePayload{Body: "rewritten"}, &actor)
	require.ErrorIs(t, err, alerts.ErrNotNoteAuthor)
}

func TestUpdateNoteAllowsTheAuthor(t *testing.T) {
	env := setup(t)
	noteID := uuid.New()
	author := uuid.New()

	env.repo.EXPECT().GetNote(gomock.Any(), noteID).
		Return(domain.AlertNote{ID: noteID, AuthorID: &author}, nil)
	env.repo.EXPECT().UpdateNote(gomock.Any(), noteID, "rewritten").
		Return(domain.AlertNote{ID: noteID, AuthorID: &author, Body: "rewritten"}, nil)

	got, err := env.service.UpdateNote(context.Background(), noteID,
		alerts.UpdateNotePayload{Body: "  rewritten  "}, &author)
	require.NoError(t, err)
	assert.Equal(t, "rewritten", got.Body)
}

func TestDeleteNoteRejectsNonAuthors(t *testing.T) {
	env := setup(t)
	noteID := uuid.New()
	author := uuid.New()

	env.repo.EXPECT().GetNote(gomock.Any(), noteID).
		Return(domain.AlertNote{ID: noteID, AuthorID: &author}, nil)
	env.repo.EXPECT().DeleteNote(gomock.Any(), gomock.Any()).Times(0)

	err := env.service.DeleteNote(context.Background(), noteID, new(uuid.New()))
	require.ErrorIs(t, err, alerts.ErrNotNoteAuthor)
}

func TestListNormalizesLimitAndReportsCursor(t *testing.T) {
	env := setup(t)
	next := "opaque-token"

	env.repo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f alerts.AlertFilter) ([]alerts.AlertListItem, *string, error) {
			assert.Equal(t, alerts.MaxLimit, f.Limit, "an over-large limit must be clamped, not honoured")
			return []alerts.AlertListItem{{ID: uuid.New()}}, &next, nil
		})
	env.repo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(42, nil)

	page, err := env.service.List(context.Background(), alerts.AlertFilter{Limit: 9999})
	require.NoError(t, err)
	assert.Equal(t, 42, page.Total)
	assert.Equal(t, &next, page.NextCursor)
	assert.Len(t, page.Entries, 1)
}

func TestListAppliesDefaultLimit(t *testing.T) {
	env := setup(t)

	env.repo.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f alerts.AlertFilter) ([]alerts.AlertListItem, *string, error) {
			assert.Equal(t, alerts.DefaultLimit, f.Limit)
			return nil, nil, nil
		})
	env.repo.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil)

	_, err := env.service.List(context.Background(), alerts.AlertFilter{})
	require.NoError(t, err)
}
