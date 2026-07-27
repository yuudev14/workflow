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
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type IncidentRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewIncidentRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *IncidentRepositoryImpl {
	return &IncidentRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *IncidentRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

const incidentAlertCount = `(SELECT count(*)::int FROM incident_alerts ia WHERE ia.incident_id = i.id) AS alert_count`

const incidentRunCount = `(SELECT count(*)::int FROM playbook_run_records prr
    WHERE prr.module_type = 'incident' AND prr.record_id = i.id) AS run_count`

// Timestamps inside the aggregate are cast to UTC because jsonb renders a bare
// `timestamp` with no offset, which Go's RFC3339 unmarshal then rejects.
const incidentRunsAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'playbook_history_id', h.id, 'playbook', p.name, 'status', h.status,
               'created_at', h.triggered_at AT TIME ZONE 'UTC')
           ORDER BY h.triggered_at DESC)
    FROM playbook_run_records prr
    JOIN playbook_history h ON h.id = prr.playbook_history_id
    LEFT JOIN playbooks p ON p.id = h.playbook_id
    WHERE prr.module_type = 'incident' AND prr.record_id = i.id
), '[]'::jsonb) AS runs`

const incidentListColumns = `i.id, i.title, i.severity, i.status, i.assignee_id,
    u.username AS assignee, i.team_id, i.tags, ` + incidentAlertCount + `, ` + incidentRunCount + `,
    i.resolved_at, i.sla_deadline, i.sla_state, i.created_at, i.updated_at`

const incidentTimelineAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', e.id, 'incident_id', e.incident_id, 'type', e.type, 'actor_id', e.actor_id,
               'actor_username', au.username, 'body', e.body, 'created_at', e.created_at AT TIME ZONE 'UTC')
           ORDER BY e.created_at, e.id)
    FROM incident_events e LEFT JOIN users au ON au.id = e.actor_id
    WHERE e.incident_id = i.id
), '[]'::jsonb) AS timeline`

const incidentNotesAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', n.id, 'incident_id', n.incident_id, 'author_id', n.author_id,
               'author_username', nu.username, 'body', n.body,
               'created_at', n.created_at AT TIME ZONE 'UTC',
               'updated_at', n.updated_at AT TIME ZONE 'UTC')
           ORDER BY n.created_at, n.id)
    FROM incident_notes n LEFT JOIN users nu ON nu.id = n.author_id
    WHERE n.incident_id = i.id
), '[]'::jsonb) AS linked_notes`

const incidentAlertsAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object(
               'id', a.id, 'title', a.title, 'severity', a.severity,
               'source_kind', a.source_kind, 'reporter', a.reporter, 'link_source', ia.source)
           ORDER BY a.created_at DESC)
    FROM incident_alerts ia JOIN alerts a ON a.id = ia.alert_id
    WHERE ia.incident_id = i.id
), '[]'::jsonb) AS linked_alerts`

type incidentDetailRow struct {
	domain.Incident
	Assignee     *string         `db:"assignee"`
	AlertCount   int             `db:"alert_count"`
	RunCount     int             `db:"run_count"`
	Timeline     json.RawMessage `db:"timeline"`
	LinkedNotes  json.RawMessage `db:"linked_notes"`
	LinkedAlerts json.RawMessage `db:"linked_alerts"`
	Runs         json.RawMessage `db:"runs"`
}

func selectIncidents(columns string) sq.SelectBuilder {
	return sq.Select(columns).
		From("incidents i").
		LeftJoin("users u ON u.id = i.assignee_id").
		PlaceholderFormat(sq.Dollar)
}

func applyIncidentFilter(stmt sq.SelectBuilder, filter incidents.IncidentFilter) (sq.SelectBuilder, error) {
	if len(filter.Status) > 0 {
		stmt = stmt.Where(sq.Eq{"i.status": filter.Status})
	}
	if len(filter.Severity) > 0 {
		stmt = stmt.Where(sq.Eq{"i.severity": filter.Severity})
	}
	if len(filter.SLAState) > 0 {
		stmt = stmt.Where(sq.Eq{"i.sla_state": filter.SLAState})
	}
	stmt = applyAssigneeFilter(stmt, "i", filter.AssigneeID, filter.Unassigned)
	if len(filter.TeamID) > 0 {
		stmt = stmt.Where(sq.Eq{"i.team_id": filter.TeamID})
	}
	if len(filter.Tags) > 0 {
		stmt = stmt.Where(sq.Expr("i.tags && ?", filter.Tags))
	}
	if filter.Open != nil && *filter.Open {
		stmt = stmt.Where(sq.Expr("i.status NOT IN ('resolved', 'closed')"))
	}

	from, to, err := filter.CreatedRange()
	if err != nil {
		return stmt, err
	}
	if from != nil {
		stmt = stmt.Where(sq.GtOrEq{"i.created_at": *from})
	}
	if to != nil {
		stmt = stmt.Where(sq.LtOrEq{"i.created_at": *to})
	}

	if term, ok := likeTerm(filter.Search); ok {
		stmt = stmt.Where(sq.Expr("i.title ILIKE ?", term))
	}
	return stmt, nil
}

func (r *IncidentRepositoryImpl) List(ctx context.Context, filter incidents.IncidentFilter) ([]incidents.IncidentListItem, *string, error) {
	stmt, err := applyIncidentFilter(selectIncidents(incidentListColumns), filter)
	if err != nil {
		return nil, nil, err
	}

	stmt, err = applyKeyset(stmt, "i", filter.Cursor)
	if err != nil {
		return nil, nil, err
	}
	stmt = orderKeyset(stmt, "i").Limit(uint64(filter.Limit + 1))
	if filter.Offset != nil {
		stmt = stmt.Offset(uint64(*filter.Offset))
	}

	rows, err := CollectRowsFromSqlizer[incidents.IncidentListItem](ctx, stmt, r.pool, r.logger)
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

func (r *IncidentRepositoryImpl) Count(ctx context.Context, filter incidents.IncidentFilter) (int, error) {
	stmt, err := applyIncidentFilter(
		sq.Select("count(*)").From("incidents i").PlaceholderFormat(sq.Dollar), filter)
	if err != nil {
		return 0, err
	}
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

func (r *IncidentRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (domain.Incident, error) {
	row, err := r.queriesFromContext(ctx).GetIncidentById(ctx, toPgUUID(id))
	if err != nil {
		return domain.Incident{}, mapNoRows(err, incidents.ErrIncidentNotFound)
	}
	return toDomainIncident(row), nil
}

func (r *IncidentRepositoryImpl) GetDetail(ctx context.Context, id uuid.UUID) (incidents.IncidentDetail, error) {
	columns := "i.*, u.username AS assignee, " + incidentAlertCount + ", " + incidentRunCount + ", " +
		incidentTimelineAggregate + ", " + incidentNotesAggregate + ", " + incidentAlertsAggregate +
		", " + incidentRunsAggregate

	rows, err := CollectRowsFromSqlizer[incidentDetailRow](
		ctx, selectIncidents(columns).Where(sq.Eq{"i.id": id}), r.pool, r.logger)
	if err != nil {
		return incidents.IncidentDetail{}, err
	}
	if len(rows) == 0 {
		return incidents.IncidentDetail{}, incidents.ErrIncidentNotFound
	}

	row := rows[0]
	detail := incidents.IncidentDetail{
		Incident:   row.Incident,
		Assignee:   row.Assignee,
		AlertCount: row.AlertCount,
		RunCount:   row.RunCount,
		IOCs:       []incidents.IOC{},
	}
	if err := json.Unmarshal(row.Runs, &detail.Runs); err != nil {
		return incidents.IncidentDetail{}, err
	}
	if err := json.Unmarshal(row.Timeline, &detail.Timeline); err != nil {
		return incidents.IncidentDetail{}, err
	}
	if err := json.Unmarshal(row.LinkedNotes, &detail.Notes); err != nil {
		return incidents.IncidentDetail{}, err
	}
	if err := json.Unmarshal(row.LinkedAlerts, &detail.LinkedAlerts); err != nil {
		return incidents.IncidentDetail{}, err
	}
	return detail, nil
}

func (r *IncidentRepositoryImpl) Create(ctx context.Context, params incidents.CreateParams) (domain.Incident, error) {
	status := string(params.Status)
	row, err := r.queriesFromContext(ctx).CreateIncident(ctx, db.CreateIncidentParams{
		Title:      params.Title,
		Severity:   db.AlertSeverity(params.Severity),
		Status:     toNullIncidentStatus(&status),
		AssigneeID: toPgUUIDPtr(params.AssigneeID),
		TeamID:     toPgUUIDPtr(params.TeamID),
		Tags:       params.Tags,
	})
	if err != nil {
		return domain.Incident{}, err
	}
	return toDomainIncident(row), nil
}

func (r *IncidentRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params incidents.UpdateParams) (domain.Incident, error) {
	row, err := r.queriesFromContext(ctx).UpdateIncident(ctx, db.UpdateIncidentParams{
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
		return domain.Incident{}, mapNoRows(err, incidents.ErrIncidentNotFound)
	}
	return toDomainIncident(row), nil
}

func (r *IncidentRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.IncidentStatus) (domain.Incident, error) {
	row, err := r.queriesFromContext(ctx).UpdateIncidentStatus(ctx, db.UpdateIncidentStatusParams{
		ID:     toPgUUID(id),
		Status: db.IncidentStatus(status),
	})
	if err != nil {
		return domain.Incident{}, mapNoRows(err, incidents.ErrIncidentNotFound)
	}
	return toDomainIncident(row), nil
}

func (r *IncidentRepositoryImpl) AppendEvent(ctx context.Context, params incidents.AppendEventParams) error {
	_, err := r.queriesFromContext(ctx).InsertIncidentEvent(ctx, db.InsertIncidentEventParams{
		IncidentID: toPgUUID(params.IncidentID),
		Type:       db.EventType(params.Type),
		ActorID:    toPgUUIDPtr(params.ActorID),
		Body:       params.Body,
	})
	return err
}

func (r *IncidentRepositoryImpl) AddNote(ctx context.Context, incidentID uuid.UUID, authorID *uuid.UUID, body string) (domain.IncidentNote, error) {
	row, err := r.queriesFromContext(ctx).InsertIncidentNote(ctx, db.InsertIncidentNoteParams{
		IncidentID: toPgUUID(incidentID),
		AuthorID:   toPgUUIDPtr(authorID),
		Body:       body,
	})
	if err != nil {
		return domain.IncidentNote{}, err
	}
	return toDomainIncidentNote(row), nil
}

func (r *IncidentRepositoryImpl) GetNote(ctx context.Context, noteID uuid.UUID) (domain.IncidentNote, error) {
	row, err := r.queriesFromContext(ctx).GetIncidentNoteById(ctx, toPgUUID(noteID))
	if err != nil {
		return domain.IncidentNote{}, mapNoRows(err, incidents.ErrNoteNotFound)
	}
	return toDomainIncidentNote(row), nil
}

func (r *IncidentRepositoryImpl) UpdateNote(ctx context.Context, noteID uuid.UUID, body string) (domain.IncidentNote, error) {
	row, err := r.queriesFromContext(ctx).UpdateIncidentNote(ctx, db.UpdateIncidentNoteParams{
		ID:   toPgUUID(noteID),
		Body: body,
	})
	if err != nil {
		return domain.IncidentNote{}, mapNoRows(err, incidents.ErrNoteNotFound)
	}
	return toDomainIncidentNote(row), nil
}

func (r *IncidentRepositoryImpl) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	affected, err := r.queriesFromContext(ctx).DeleteIncidentNote(ctx, toPgUUID(noteID))
	if err != nil {
		return err
	}
	if affected == 0 {
		return incidents.ErrNoteNotFound
	}
	return nil
}

func (r *IncidentRepositoryImpl) ListNotes(ctx context.Context, incidentID uuid.UUID) ([]domain.IncidentNote, error) {
	rows, err := r.queriesFromContext(ctx).ListIncidentNotes(ctx, toPgUUID(incidentID))
	if err != nil {
		return nil, err
	}

	notes := make([]domain.IncidentNote, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, domain.IncidentNote{
			ID:             fromPgUUID(row.ID),
			IncidentID:     fromPgUUID(row.IncidentID),
			AuthorID:       fromPgUUIDPtr(row.AuthorID),
			AuthorUsername: fromPgText(row.AuthorUsername),
			Body:           row.Body,
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		})
	}
	return notes, nil
}

// ON CONFLICT DO NOTHING means zero rows affected is a repeat link, not a
// failure - the caller uses that to skip a duplicate timeline entry.
func (r *IncidentRepositoryImpl) LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID, source domain.LinkSource) (bool, error) {
	affected, err := r.queriesFromContext(ctx).InsertIncidentAlert(ctx, db.InsertIncidentAlertParams{
		IncidentID: toPgUUID(incidentID),
		AlertID:    toPgUUID(alertID),
		Source:     db.LinkSource(source),
	})
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *IncidentRepositoryImpl) UnlinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error {
	affected, err := r.queriesFromContext(ctx).DeleteIncidentAlert(ctx, db.DeleteIncidentAlertParams{
		IncidentID: toPgUUID(incidentID),
		AlertID:    toPgUUID(alertID),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return incidents.ErrAlertNotLinked
	}
	return nil
}

func (r *IncidentRepositoryImpl) Summary(ctx context.Context, rng types.ResolvedRange) (incidents.IncidentsSummary, error) {
	summary := incidents.IncidentsSummary{
		StatusMix:   []incidents.StatusBucket{},
		SeverityMix: []incidents.SeverityBucket{},
		MTTRTrend:   []incidents.MTTRPoint{},
		SLAAtRisk:   []incidents.SLARisk{},
		Range:       rng,
	}

	openFilter := sq.Expr("i.status NOT IN ('resolved', 'closed')")

	openTotal, err := CollectOneScalarFromSqlizer[int](ctx,
		sq.Select("count(*)").From("incidents i").Where(openFilter).PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.OpenTotal = openTotal

	statusMix, err := CollectRowsFromSqlizer[incidents.StatusBucket](ctx,
		sq.Select("i.status, count(*)::int AS count").From("incidents i").
			Where(openFilter).GroupBy("i.status").OrderBy("i.status").
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.StatusMix = statusMix

	severityMix, err := CollectRowsFromSqlizer[incidents.SeverityBucket](ctx,
		sq.Select("i.severity, count(*)::int AS count").From("incidents i").
			Where(openFilter).GroupBy("i.severity").OrderBy("i.severity").
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.SeverityMix = severityMix

	// SLA deadlines are never stamped yet (#14), so this is empty in practice -
	// the query is here so it lights up the day a policy engine sets them.
	slaAtRisk, err := CollectRowsFromSqlizer[incidents.SLARisk](ctx,
		sq.Select("i.id, i.title, i.sla_deadline, (i.sla_state = 'breached') AS breached").
			From("incidents i").
			Where(openFilter).
			Where(sq.Expr("i.sla_deadline IS NOT NULL")).
			OrderBy("i.sla_deadline").
			Limit(5).
			PlaceholderFormat(sq.Dollar),
		r.pool, r.logger)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.SLAAtRisk = slaAtRisk

	mttr, err := r.mttrTrend(ctx, rng)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.MTTRTrend = mttr

	prevFrom, prevTo := rng.Previous()
	created, err := r.windowCount(ctx, "created_at", rng, prevFrom, prevTo)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.Created = created

	resolved, err := r.windowCount(ctx, "resolved_at", rng, prevFrom, prevTo)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.Resolved = resolved

	mttrWindow, err := r.mttrWindow(ctx, rng, prevFrom, prevTo)
	if err != nil {
		return incidents.IncidentsSummary{}, err
	}
	summary.MTTRSeconds = mttrWindow

	return summary, nil
}

// A bucket with no resolutions reports 0 rather than dropping out and shifting
// every later point left.
func (r *IncidentRepositoryImpl) mttrTrend(ctx context.Context, rng types.ResolvedRange) ([]incidents.MTTRPoint, error) {
	const q = `
        SELECT d.bucket_start AT TIME ZONE 'UTC' AS bucket_start,
               COALESCE(c.avg_seconds, 0)::int AS avg_seconds
        FROM generate_series(
                 date_trunc($3, $1::timestamptz), $2::timestamptz, ('1 ' || $3)::interval
             ) AS d(bucket_start)
        LEFT JOIN (
            SELECT date_trunc($3, resolved_at) AS bucket_start,
                   avg(EXTRACT(EPOCH FROM (resolved_at - created_at))) AS avg_seconds
            FROM incidents
            WHERE resolved_at IS NOT NULL AND resolved_at >= $1 AND resolved_at <= $2
            GROUP BY 1
        ) c ON c.bucket_start = d.bucket_start
        ORDER BY d.bucket_start`

	rows, err := r.pool.Query(ctx, q, rng.From, rng.To, rng.Bucket)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]incidents.MTTRPoint, 0, 32)
	for rows.Next() {
		var p incidents.MTTRPoint
		if err := rows.Scan(&p.BucketStart, &p.AvgSeconds); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// windowCount counts rows whose column falls in the selected range beside the
// same count for the preceding range - the only baseline that makes the delta
// on a KPI card mean anything.
func (r *IncidentRepositoryImpl) windowCount(ctx context.Context, column string, rng types.ResolvedRange, prevFrom, prevTo time.Time) (types.WindowCount, error) {
	q := fmt.Sprintf(`
        SELECT
            count(*) FILTER (WHERE %[1]s >= $1 AND %[1]s <= $2)::int AS current,
            count(*) FILTER (WHERE %[1]s >= $3 AND %[1]s < $4)::int AS previous
        FROM incidents`, column)

	var out types.WindowCount
	err := r.pool.QueryRow(ctx, q, rng.From, rng.To, prevFrom, prevTo).Scan(&out.Current, &out.Previous)
	return out, err
}

func (r *IncidentRepositoryImpl) mttrWindow(ctx context.Context, rng types.ResolvedRange, prevFrom, prevTo time.Time) (types.WindowCount, error) {
	const q = `
        SELECT
            COALESCE(avg(EXTRACT(EPOCH FROM (resolved_at - created_at)))
                     FILTER (WHERE resolved_at >= $1 AND resolved_at <= $2), 0)::int AS current,
            COALESCE(avg(EXTRACT(EPOCH FROM (resolved_at - created_at)))
                     FILTER (WHERE resolved_at >= $3 AND resolved_at < $4), 0)::int AS previous
        FROM incidents WHERE resolved_at IS NOT NULL`

	var out types.WindowCount
	err := r.pool.QueryRow(ctx, q, rng.From, rng.To, prevFrom, prevTo).Scan(&out.Current, &out.Previous)
	return out, err
}

func toDomainIncident(row db.Incident) domain.Incident {
	return domain.Incident{
		ID:            fromPgUUID(row.ID),
		Title:         row.Title,
		Severity:      domain.AlertSeverity(row.Severity),
		Status:        domain.IncidentStatus(row.Status),
		AssigneeID:    fromPgUUIDPtr(row.AssigneeID),
		TeamID:        fromPgUUIDPtr(row.TeamID),
		Tags:          row.Tags,
		ResolvedAt:    fromPgTimestampPtr(row.ResolvedAt),
		SLADeadline:   fromPgTimestampPtr(row.SlaDeadline),
		SLABreachedAt: fromPgTimestampPtr(row.SlaBreachedAt),
		SLAState:      domain.SLAState(row.SlaState),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func toDomainIncidentNote(row db.IncidentNote) domain.IncidentNote {
	return domain.IncidentNote{
		ID:         fromPgUUID(row.ID),
		IncidentID: fromPgUUID(row.IncidentID),
		AuthorID:   fromPgUUIDPtr(row.AuthorID),
		Body:       row.Body,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
