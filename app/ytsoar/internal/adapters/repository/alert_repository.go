package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type AlertRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewAlertRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *AlertRepositoryImpl {
	return &AlertRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *AlertRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

// payload and triage are detail-only: a queue page of 50 would otherwise drag
// 50 raw event documents across the wire.
const alertRunCount = `(SELECT count(*)::int FROM playbook_run_records prr
    WHERE prr.module_type = 'alert' AND prr.record_id = a.id) AS run_count`

const alertListColumns = `a.id, a.title, a.severity, a.status, a.source_kind, a.reporter,
    a.assignee_id, u.username AS assignee, a.team_id, a.tags, a.dedup_count, a.last_seen,
    a.triaged_at, a.sla_deadline, a.sla_state, a.created_at, a.updated_at, ` + alertRunCount

// Timestamps inside the aggregate are cast to UTC because jsonb renders a bare
// `timestamp` with no offset, which Go's RFC3339 unmarshal then rejects.
const alertRunsAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'playbook_history_id', h.id, 'playbook_id', h.playbook_id,
               'playbook', p.name, 'status', h.status,
               'trigger_type', h.trigger_type, 'triggered_by', tu.username,
               'created_at', h.triggered_at AT TIME ZONE 'UTC')
           ORDER BY h.triggered_at DESC)
    FROM playbook_run_records prr
    JOIN playbook_history h ON h.id = prr.playbook_history_id
    LEFT JOIN playbooks p ON p.id = h.playbook_id
    LEFT JOIN users tu ON tu.id = h.triggered_by
    WHERE prr.module_type = 'alert' AND prr.record_id = a.id
), '[]'::jsonb) AS runs`

// The timestamps are cast to UTC because jsonb renders a bare `timestamp` with
// no offset, which Go's RFC3339 unmarshal then rejects. The top-level columns
// are unaffected - pgx scans those natively.
const alertTimelineAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', e.id, 'alert_id', e.alert_id, 'type', e.type, 'actor_id', e.actor_id,
               'actor_username', au.username, 'body', e.body, 'created_at', e.created_at AT TIME ZONE 'UTC')
           ORDER BY e.created_at, e.id)
    FROM alert_events e LEFT JOIN users au ON au.id = e.actor_id
    WHERE e.alert_id = a.id
), '[]'::jsonb) AS timeline`

const alertNotesAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', n.id, 'alert_id', n.alert_id, 'author_id', n.author_id,
               'author_username', nu.username, 'body', n.body,
               'created_at', n.created_at AT TIME ZONE 'UTC',
               'updated_at', n.updated_at AT TIME ZONE 'UTC')
           ORDER BY n.created_at, n.id)
    FROM alert_notes n LEFT JOIN users nu ON nu.id = n.author_id
    WHERE n.alert_id = a.id
), '[]'::jsonb) AS notes`

const alertIncidentsAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', i.id, 'title', i.title, 'status', i.status,
               'severity', i.severity, 'link_source', ia.source)
           ORDER BY i.created_at DESC)
    FROM incident_alerts ia JOIN incidents i ON i.id = ia.incident_id
    WHERE ia.alert_id = a.id
), '[]'::jsonb) AS linked_incidents`

type alertDetailRow struct {
	domain.Alert
	Assignee        *string         `db:"assignee"`
	Timeline        json.RawMessage `db:"timeline"`
	Notes           json.RawMessage `db:"notes"`
	LinkedIncidents json.RawMessage `db:"linked_incidents"`
	RunCount        int             `db:"run_count"`
	Runs            json.RawMessage `db:"runs"`
}

func selectAlerts(columns string) sq.SelectBuilder {
	return sq.Select(columns).
		From("alerts a").
		LeftJoin("users u ON u.id = a.assignee_id").
		PlaceholderFormat(sq.Dollar)
}

func applyAlertFilter(stmt sq.SelectBuilder, filter alerts.AlertFilter) (sq.SelectBuilder, error) {
	if len(filter.Status) > 0 {
		stmt = stmt.Where(sq.Eq{"a.status": filter.Status})
	}
	if len(filter.Severity) > 0 {
		stmt = stmt.Where(sq.Eq{"a.severity": filter.Severity})
	}
	if len(filter.SourceKind) > 0 {
		stmt = stmt.Where(sq.Eq{"a.source_kind": filter.SourceKind})
	}
	if len(filter.SLAState) > 0 {
		stmt = stmt.Where(sq.Eq{"a.sla_state": filter.SLAState})
	}
	stmt = applyAssigneeFilter(stmt, "a", filter.AssigneeID, filter.Unassigned)
	if len(filter.TeamID) > 0 {
		stmt = stmt.Where(sq.Eq{"a.team_id": filter.TeamID})
	}
	if len(filter.Tags) > 0 {
		stmt = stmt.Where(sq.Expr("a.tags && ?", filter.Tags))
	}
	if filter.DedupMin != nil {
		stmt = stmt.Where(sq.GtOrEq{"a.dedup_count": *filter.DedupMin})
	}
	if filter.Triaged != nil {
		if *filter.Triaged {
			stmt = stmt.Where(sq.Expr("a.triaged_at IS NOT NULL"))
		} else {
			stmt = stmt.Where(sq.Expr("a.triaged_at IS NULL"))
		}
	}

	from, to, err := filter.CreatedRange()
	if err != nil {
		return stmt, err
	}
	if from != nil {
		stmt = stmt.Where(sq.GtOrEq{"a.created_at": *from})
	}
	if to != nil {
		stmt = stmt.Where(sq.LtOrEq{"a.created_at": *to})
	}

	if term, ok := likeTerm(filter.Search); ok {
		stmt = stmt.Where(sq.Expr("(a.title ILIKE ? OR a.reporter ILIKE ?)", term, term))
	}
	return stmt, nil
}

// Fetching limit+1 is what decides whether a next page exists, without a second
// query.
func (r *AlertRepositoryImpl) List(ctx context.Context, filter alerts.AlertFilter) ([]alerts.AlertListItem, *string, error) {
	stmt, err := applyAlertFilter(selectAlerts(alertListColumns), filter)
	if err != nil {
		return nil, nil, err
	}

	stmt, err = applyKeyset(stmt, "a", filter.Cursor)
	if err != nil {
		return nil, nil, err
	}
	stmt = orderKeyset(stmt, "a").Limit(uint64(filter.Limit + 1))
	if filter.Offset != nil {
		stmt = stmt.Offset(uint64(*filter.Offset))
	}

	rows, err := CollectRowsFromSqlizer[alerts.AlertListItem](ctx, stmt, r.pool, r.logger)
	if err != nil {
		return nil, nil, err
	}

	hasMore := len(rows) > filter.Limit
	if hasMore {
		rows = rows[:filter.Limit]
	}

	// Offset mode returns no cursor: handing back both would invite a client to
	// interleave the two paging modes on one result set.
	if !hasMore || filter.Offset != nil {
		return rows, nil, nil
	}

	last := rows[len(rows)-1]
	next, err := encodeCursor("created_at", last.CreatedAt, last.ID)
	if err != nil {
		return nil, nil, err
	}
	return rows, &next, nil
}

func (r *AlertRepositoryImpl) Count(ctx context.Context, filter alerts.AlertFilter) (int, error) {
	stmt, err := applyAlertFilter(
		sq.Select("count(*)").From("alerts a").PlaceholderFormat(sq.Dollar), filter)
	if err != nil {
		return 0, err
	}
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

func (r *AlertRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (domain.Alert, error) {
	row, err := r.queriesFromContext(ctx).GetAlertById(ctx, toPgUUID(id))
	if err != nil {
		return domain.Alert{}, mapNoRows(err, alerts.ErrAlertNotFound)
	}
	return toDomainAlert(row), nil
}

func (r *AlertRepositoryImpl) GetDetail(ctx context.Context, id uuid.UUID) (alerts.AlertDetail, error) {
	columns := "a.*, u.username AS assignee, " +
		alertTimelineAggregate + ", " + alertNotesAggregate + ", " + alertIncidentsAggregate +
		", " + alertRunCount + ", " + alertRunsAggregate

	rows, err := CollectRowsFromSqlizer[alertDetailRow](
		ctx, selectAlerts(columns).Where(sq.Eq{"a.id": id}), r.pool, r.logger)
	if err != nil {
		return alerts.AlertDetail{}, err
	}
	if len(rows) == 0 {
		return alerts.AlertDetail{}, alerts.ErrAlertNotFound
	}

	row := rows[0]
	detail := alerts.AlertDetail{
		Alert:         row.Alert,
		Assignee:      row.Assignee,
		RelatedAlerts: []alerts.RelatedAlert{},
		RunCount:      row.RunCount,
	}
	if err := json.Unmarshal(row.Runs, &detail.Runs); err != nil {
		return alerts.AlertDetail{}, err
	}
	if err := json.Unmarshal(row.Timeline, &detail.Timeline); err != nil {
		return alerts.AlertDetail{}, err
	}
	if err := json.Unmarshal(row.Notes, &detail.Notes); err != nil {
		return alerts.AlertDetail{}, err
	}
	if err := json.Unmarshal(row.LinkedIncidents, &detail.LinkedIncidents); err != nil {
		return alerts.AlertDetail{}, err
	}
	return detail, nil
}

func (r *AlertRepositoryImpl) Upsert(ctx context.Context, params alerts.UpsertParams) (domain.Alert, bool, error) {
	row, err := r.queriesFromContext(ctx).UpsertAlert(ctx, db.UpsertAlertParams{
		Title:       params.Title,
		Severity:    db.AlertSeverity(params.Severity),
		SourceKind:  db.SourceKind(params.SourceKind),
		Reporter:    toPgText(params.Reporter),
		AssigneeID:  toPgUUIDPtr(params.AssigneeID),
		TeamID:      toPgUUIDPtr(params.TeamID),
		Payload:     params.Payload,
		Tags:        params.Tags,
		Fingerprint: params.Fingerprint,
		CreatedAt:   toPgTimestampPtr(params.CreatedAt),
	})
	if err != nil {
		return domain.Alert{}, false, err
	}
	return toDomainAlertUpsert(row), row.Inserted, nil
}

func (r *AlertRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params alerts.UpdateParams) (domain.Alert, error) {
	row, err := r.queriesFromContext(ctx).UpdateAlert(ctx, db.UpdateAlertParams{
		ID:            toPgUUID(id),
		SeveritySet:   params.Severity.Set,
		Severity:      toNullAlertSeverity(params.Severity),
		AssigneeIDSet: params.AssigneeID.Set,
		AssigneeID:    toPgUUIDFromNullable(params.AssigneeID),
		TeamIDSet:     params.TeamID.Set,
		TeamID:        toPgUUIDFromNullable(params.TeamID),
		TagsSet:       params.Tags.Set,
		Tags:          fromNullableStrings(params.Tags),
	})
	if err != nil {
		return domain.Alert{}, mapNoRows(err, alerts.ErrAlertNotFound)
	}
	return toDomainAlert(row), nil
}

func (r *AlertRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AlertStatus, closureNote *string) (domain.Alert, error) {
	row, err := r.queriesFromContext(ctx).UpdateAlertStatus(ctx, db.UpdateAlertStatusParams{
		ID:          toPgUUID(id),
		Status:      db.AlertStatus(status),
		ClosureNote: toPgText(closureNote),
	})
	if err != nil {
		return domain.Alert{}, mapNoRows(err, alerts.ErrAlertNotFound)
	}
	return toDomainAlert(row), nil
}

func (r *AlertRepositoryImpl) AppendEvent(ctx context.Context, params alerts.AppendEventParams) error {
	_, err := r.queriesFromContext(ctx).InsertAlertEvent(ctx, db.InsertAlertEventParams{
		AlertID: toPgUUID(params.AlertID),
		Type:    db.EventType(params.Type),
		ActorID: toPgUUIDPtr(params.ActorID),
		Body:    params.Body,
	})
	return err
}

// AppendAlertEvent satisfies incidents.AlertTimeline so linking can write the
// other half of the story onto the alert.
func (r *AlertRepositoryImpl) AppendAlertEvent(ctx context.Context, alertID uuid.UUID, eventType domain.EventType, actorID *uuid.UUID, body []byte) error {
	return r.AppendEvent(ctx, alerts.AppendEventParams{
		AlertID: alertID,
		Type:    eventType,
		ActorID: actorID,
		Body:    body,
	})
}

func (r *AlertRepositoryImpl) AddNote(ctx context.Context, alertID uuid.UUID, authorID *uuid.UUID, body string) (domain.AlertNote, error) {
	row, err := r.queriesFromContext(ctx).InsertAlertNote(ctx, db.InsertAlertNoteParams{
		AlertID:  toPgUUID(alertID),
		AuthorID: toPgUUIDPtr(authorID),
		Body:     body,
	})
	if err != nil {
		return domain.AlertNote{}, err
	}
	return toDomainAlertNote(row), nil
}

func (r *AlertRepositoryImpl) GetNote(ctx context.Context, noteID uuid.UUID) (domain.AlertNote, error) {
	row, err := r.queriesFromContext(ctx).GetAlertNoteById(ctx, toPgUUID(noteID))
	if err != nil {
		return domain.AlertNote{}, mapNoRows(err, alerts.ErrNoteNotFound)
	}
	return toDomainAlertNote(row), nil
}

func (r *AlertRepositoryImpl) UpdateNote(ctx context.Context, noteID uuid.UUID, body string) (domain.AlertNote, error) {
	row, err := r.queriesFromContext(ctx).UpdateAlertNote(ctx, db.UpdateAlertNoteParams{
		ID:   toPgUUID(noteID),
		Body: body,
	})
	if err != nil {
		return domain.AlertNote{}, mapNoRows(err, alerts.ErrNoteNotFound)
	}
	return toDomainAlertNote(row), nil
}

func (r *AlertRepositoryImpl) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	affected, err := r.queriesFromContext(ctx).DeleteAlertNote(ctx, toPgUUID(noteID))
	if err != nil {
		return err
	}
	if affected == 0 {
		return alerts.ErrNoteNotFound
	}
	return nil
}

func (r *AlertRepositoryImpl) ListNotes(ctx context.Context, alertID uuid.UUID) ([]domain.AlertNote, error) {
	rows, err := r.queriesFromContext(ctx).ListAlertNotes(ctx, toPgUUID(alertID))
	if err != nil {
		return nil, err
	}

	notes := make([]domain.AlertNote, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, domain.AlertNote{
			ID:             fromPgUUID(row.ID),
			AlertID:        fromPgUUID(row.AlertID),
			AuthorID:       fromPgUUIDPtr(row.AuthorID),
			AuthorUsername: fromPgText(row.AuthorUsername),
			Body:           row.Body,
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		})
	}
	return notes, nil
}

// Every bucket is a live GROUP BY over the open partial index rather than a
// rollup table: the open set stays small even when alerts does not, and a
// counter would drift.
func (r *AlertRepositoryImpl) Summary(ctx context.Context, rng types.ResolvedRange) (alerts.AlertsSummary, error) {
	summary := alerts.AlertsSummary{
		BySeverity:   []alerts.SeverityBucket{},
		BySource:     []alerts.SourceBucket{},
		TopPlaybooks: []alerts.PlaybookSuccess{},
		Volume:       []alerts.VolumePoint{},
		Range:        rng,
	}

	openFilter := sq.Expr("a.status IN ('new', 'investigating')")

	total, err := CollectOneScalarFromSqlizer[int](ctx,
		sq.Select("count(*)").From("alerts a").Where(openFilter).PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.Total = total

	bySeverity, err := CollectRowsFromSqlizer[alerts.SeverityBucket](ctx,
		sq.Select("a.severity, count(*)::int AS count").From("alerts a").
			Where(openFilter).GroupBy("a.severity").OrderBy("a.severity").
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.BySeverity = bySeverity

	bySource, err := CollectRowsFromSqlizer[alerts.SourceBucket](ctx,
		sq.Select("a.source_kind, count(*)::int AS count").From("alerts a").
			Where(openFilter).GroupBy("a.source_kind").OrderBy("count(*) DESC").
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.BySource = bySource

	// Scoped to alert runs so the "share of that playbook's runs on alerts"
	// label on the dashboard is actually true.
	topPlaybooks, err := CollectRowsFromSqlizer[alerts.PlaybookSuccess](ctx,
		sq.Select(`p.name AS label,
            (count(*) FILTER (WHERE h.status = 'success'))::float / count(*)::float AS success_rate`).
			From("playbook_history h").
			Join("playbooks p ON p.id = h.playbook_id").
			Where(sq.Expr(`EXISTS (
                SELECT 1 FROM playbook_run_records prr
                WHERE prr.playbook_history_id = h.id AND prr.module_type = ?)`, domain.ModuleEventAlert)).
			Where(sq.GtOrEq{"h.triggered_at": rng.From}).
			Where(sq.LtOrEq{"h.triggered_at": rng.To}).
			GroupBy("p.name").
			OrderBy("count(*) DESC").
			Limit(5).
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.TopPlaybooks = topPlaybooks

	volume, err := r.volume(ctx, rng)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.Volume = volume

	prevFrom, prevTo := rng.Previous()
	created, err := r.windowCount(ctx, "created_at", rng, prevFrom, prevTo)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.Created = created

	resolved, err := r.resolvedWindow(ctx, rng, prevFrom, prevTo)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.Resolved = resolved

	mttt, err := r.mtttWindow(ctx, rng, prevFrom, prevTo)
	if err != nil {
		return alerts.AlertsSummary{}, err
	}
	summary.MTTTSeconds = mttt

	return summary, nil
}

// generate_series supplies the empty buckets; a plain GROUP BY drops them, and
// the chart would then compress a quiet week into a spike.
func (r *AlertRepositoryImpl) volume(ctx context.Context, rng types.ResolvedRange) ([]alerts.VolumePoint, error) {
	const q = `
        SELECT d.bucket_start AT TIME ZONE 'UTC' AS bucket_start, COALESCE(c.count, 0)::int AS count
        FROM generate_series(
                 date_trunc($3, $1::timestamptz), $2::timestamptz, ('1 ' || $3)::interval
             ) AS d(bucket_start)
        LEFT JOIN (
            SELECT date_trunc($3, created_at) AS bucket_start, count(*) AS count
            FROM alerts
            WHERE created_at >= $1 AND created_at <= $2
            GROUP BY 1
        ) c ON c.bucket_start = d.bucket_start
        ORDER BY d.bucket_start`

	rows, err := r.pool.Query(ctx, q, rng.From, rng.To, rng.Bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]alerts.VolumePoint, 0, 32)
	for rows.Next() {
		var p alerts.VolumePoint
		if err := rows.Scan(&p.BucketStart, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// windowCount counts rows whose column falls in the selected range beside the
// same count for the preceding range - the only baseline that makes the delta
// on a KPI card mean anything.
func (r *AlertRepositoryImpl) windowCount(ctx context.Context, column string, rng types.ResolvedRange, prevFrom, prevTo time.Time) (types.WindowCount, error) {
	q := fmt.Sprintf(`
        SELECT
            count(*) FILTER (WHERE %[1]s >= $1 AND %[1]s <= $2)::int AS current,
            count(*) FILTER (WHERE %[1]s >= $3 AND %[1]s < $4)::int AS previous
        FROM alerts`, column)

	var out types.WindowCount
	err := r.pool.QueryRow(ctx, q, rng.From, rng.To, prevFrom, prevTo).Scan(&out.Current, &out.Previous)
	return out, err
}

// alerts has no resolved_at column, so the timeline is the authority on when an
// alert was closed out. Counting the event rather than the current status is
// also what makes the previous window answerable at all - a row that was
// resolved and later reopened must still count in the window it was resolved in.
func (r *AlertRepositoryImpl) resolvedWindow(ctx context.Context, rng types.ResolvedRange, prevFrom, prevTo time.Time) (types.WindowCount, error) {
	const q = `
        SELECT
            count(DISTINCT e.alert_id) FILTER (WHERE e.created_at >= $1 AND e.created_at <= $2)::int AS current,
            count(DISTINCT e.alert_id) FILTER (WHERE e.created_at >= $3 AND e.created_at < $4)::int AS previous
        FROM alert_events e
        WHERE e.type = 'status_changed' AND e.body->>'to' IN ('resolved', 'falsepos', 'closed')`

	var out types.WindowCount
	err := r.pool.QueryRow(ctx, q, rng.From, rng.To, prevFrom, prevTo).Scan(&out.Current, &out.Previous)
	return out, err
}

func (r *AlertRepositoryImpl) mtttWindow(ctx context.Context, rng types.ResolvedRange, prevFrom, prevTo time.Time) (types.WindowCount, error) {
	const q = `
        SELECT
            COALESCE(avg(EXTRACT(EPOCH FROM (triaged_at - created_at)))
                     FILTER (WHERE triaged_at >= $1 AND triaged_at <= $2), 0)::int AS current,
            COALESCE(avg(EXTRACT(EPOCH FROM (triaged_at - created_at)))
                     FILTER (WHERE triaged_at >= $3 AND triaged_at < $4), 0)::int AS previous
        FROM alerts WHERE triaged_at IS NOT NULL`

	var out types.WindowCount
	err := r.pool.QueryRow(ctx, q, rng.From, rng.To, prevFrom, prevTo).Scan(&out.Current, &out.Previous)
	return out, err
}

func toDomainAlert(row db.Alert) domain.Alert {
	return domain.Alert{
		ID:            fromPgUUID(row.ID),
		Title:         row.Title,
		Severity:      domain.AlertSeverity(row.Severity),
		Status:        domain.AlertStatus(row.Status),
		SourceKind:    domain.SourceKind(row.SourceKind),
		Reporter:      fromPgText(row.Reporter),
		AssigneeID:    fromPgUUIDPtr(row.AssigneeID),
		TeamID:        fromPgUUIDPtr(row.TeamID),
		Payload:       row.Payload,
		Tags:          row.Tags,
		Triage:        row.Triage,
		ClosureNote:   fromPgText(row.ClosureNote),
		Fingerprint:   row.Fingerprint,
		DedupCount:    row.DedupCount,
		LastSeen:      row.LastSeen.Time,
		TriagedAt:     fromPgTimestampPtr(row.TriagedAt),
		SLADeadline:   fromPgTimestampPtr(row.SlaDeadline),
		SLABreachedAt: fromPgTimestampPtr(row.SlaBreachedAt),
		SLAState:      domain.SLAState(row.SlaState),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func toDomainAlertUpsert(row db.UpsertAlertRow) domain.Alert {
	return toDomainAlert(db.Alert{
		ID:            row.ID,
		Title:         row.Title,
		Severity:      row.Severity,
		Status:        row.Status,
		SourceKind:    row.SourceKind,
		Reporter:      row.Reporter,
		AssigneeID:    row.AssigneeID,
		TeamID:        row.TeamID,
		Payload:       row.Payload,
		Tags:          row.Tags,
		Triage:        row.Triage,
		ClosureNote:   row.ClosureNote,
		Fingerprint:   row.Fingerprint,
		DedupCount:    row.DedupCount,
		LastSeen:      row.LastSeen,
		TriagedAt:     row.TriagedAt,
		SlaDeadline:   row.SlaDeadline,
		SlaBreachedAt: row.SlaBreachedAt,
		SlaState:      row.SlaState,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	})
}

func toDomainAlertNote(row db.AlertNote) domain.AlertNote {
	return domain.AlertNote{
		ID:        fromPgUUID(row.ID),
		AlertID:   fromPgUUID(row.AlertID),
		AuthorID:  fromPgUUIDPtr(row.AuthorID),
		Body:      row.Body,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
