package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
	"github.com/yuudev14/ytsoar/internal/utils"
)

type PlaybookRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewPlaybookRepository(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *PlaybookRepositoryImpl {
	return &PlaybookRepositoryImpl{logger: log, q: q, pool: pool}
}

func (w *PlaybookRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return w.q.WithTx(tx)
	}
	return w.q
}

func toDomainPlaybook(row db.Playbook) domain.Playbooks {
	return domain.Playbooks{
		ID:                fromPgUUID(row.ID),
		Name:              fromPgTextString(row.Name),
		Description:       fromPgText(row.Description),
		TriggerType:       fromNullTriggerType(row.TriggerType),
		TriggerParameters: row.TriggerParameters,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func toDomainPlaybookHistory(row db.PlaybookHistory) domain.PlaybookHistory {
	var result any
	if len(row.Result) > 0 {
		json.Unmarshal(row.Result, &result)
	}
	return domain.PlaybookHistory{
		ID:          fromPgUUID(row.ID),
		PlaybookID:  fromPgUUID(row.PlaybookID),
		Status:      string(row.Status),
		Error:       fromPgText(row.Error),
		Result:      result,
		TriggeredAt: row.TriggeredAt.Time,
		Edges:       row.Edges,
		TriggerType: fromNullTriggerType(row.TriggerType),
		TriggeredBy: fromPgUUIDPtr(row.TriggeredBy),
		Input:       row.Input,
	}
}

// GetPlaybooks implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybooks(ctx context.Context, offset int, limit int, filter playbooks.PlaybookFilter) ([]domain.Playbooks, error) {
	stmt := sq.Select("*").From("playbooks").
		PlaceholderFormat(sq.Dollar).
		OrderBy("updated_at DESC").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	if filter.Name != nil {
		stmt = stmt.Where(sq.Expr("name ILIKE ?", fmt.Sprint("%", *filter.Name, "%")))
	}

	return CollectRowsFromSqlizer[domain.Playbooks](ctx, stmt, w.pool, w.logger)
}

// GetPlaybooksCount implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybooksCount(ctx context.Context, filter playbooks.PlaybookFilter) (int, error) {
	stmt := sq.Select("count(*)").From("playbooks").
		PlaceholderFormat(sq.Dollar)

	if filter.Name != nil {
		stmt = stmt.Where(sq.Expr("name ILIKE ?", fmt.Sprint("%", *filter.Name, "%")))
	}

	return CollectOneScalarFromSqlizer[int](ctx, stmt, w.pool, w.logger)
}

// Summary implements playbooks.PlaybookRepository. Each figure is paired with the
// preceding window of equal length so the UI can render an honest delta.
//
// Playbooks.previous is deliberately "how many existed before the window", not
// "how many were created in the previous window": the card shows a running total,
// so its delta is the growth inside the selected range.
func (w *PlaybookRepositoryImpl) Summary(ctx context.Context, rng types.ResolvedRange) (playbooks.PlaybooksSummary, error) {
	summary := playbooks.PlaybooksSummary{Range: rng}
	prevFrom, prevTo := rng.Previous()

	const playbookCounts = `
        SELECT
            count(*) FILTER (WHERE created_at <= $2)::int AS current,
            count(*) FILTER (WHERE created_at < $1)::int AS previous
        FROM playbooks`
	if err := w.pool.QueryRow(ctx, playbookCounts, rng.From, rng.To).
		Scan(&summary.Playbooks.Current, &summary.Playbooks.Previous); err != nil {
		return playbooks.PlaybooksSummary{}, err
	}

	const runCounts = `
        SELECT
            count(*) FILTER (WHERE in_current)::int AS runs_current,
            count(*) FILTER (WHERE in_previous)::int AS runs_previous,
            count(*) FILTER (WHERE in_current AND status = 'failed')::int AS failed_current,
            count(*) FILTER (WHERE in_previous AND status = 'failed')::int AS failed_previous,
            COALESCE(count(*) FILTER (WHERE in_current AND status = 'success')::float
                     / NULLIF(count(*) FILTER (WHERE in_current), 0), 0) AS rate_current,
            COALESCE(count(*) FILTER (WHERE in_previous AND status = 'success')::float
                     / NULLIF(count(*) FILTER (WHERE in_previous), 0), 0) AS rate_previous
        FROM (
            SELECT status,
                   triggered_at >= $1 AND triggered_at <= $2 AS in_current,
                   triggered_at >= $3 AND triggered_at <  $4 AS in_previous
            FROM playbook_history
        ) h`
	if err := w.pool.QueryRow(ctx, runCounts, rng.From, rng.To, prevFrom, prevTo).Scan(
		&summary.Runs.Current, &summary.Runs.Previous,
		&summary.Failed.Current, &summary.Failed.Previous,
		&summary.SuccessRate.Current, &summary.SuccessRate.Previous,
	); err != nil {
		return playbooks.PlaybooksSummary{}, err
	}

	return summary, nil
}

// GetPlaybookHistory implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistory(ctx context.Context, offset int, limit int, filter playbooks.PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error) {
	stmt := sq.Select("playbook_history.*, to_jsonb(playbooks) AS playbook_data").
		From("playbook_history").
		Join("playbooks ON playbooks.id = playbook_history.playbook_id").
		PlaceholderFormat(sq.Dollar).
		OrderBy("triggered_at DESC").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	stmt = applyPlaybookHistoryFilter(stmt, filter)

	return CollectRowsFromSqlizer[domain.PlaybookHistoryResponse](ctx, stmt, w.pool, w.logger)
}

// GetPlaybookHistoryCount implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistoryCount(ctx context.Context, filter playbooks.PlaybookHistoryFilter) (int, error) {
	stmt := sq.Select("count(playbook_history.*)").
		From("playbook_history").
		Join("playbooks ON playbooks.id = playbook_history.playbook_id").
		PlaceholderFormat(sq.Dollar)

	stmt = applyPlaybookHistoryFilter(stmt, filter)

	return CollectOneScalarFromSqlizer[int](ctx, stmt, w.pool, w.logger)
}

// Shared by the list and the count so `total` can never disagree with the page
// it describes - previously the count applied only Name.
func applyPlaybookHistoryFilter(stmt sq.SelectBuilder, filter playbooks.PlaybookHistoryFilter) sq.SelectBuilder {
	if term, ok := likeTerm(filter.Name); ok {
		stmt = stmt.Where(sq.Expr("playbooks.name ILIKE ?", term))
	}
	if filter.PlaybookID != nil {
		stmt = stmt.Where(sq.Eq{"playbook_history.playbook_id": *filter.PlaybookID})
	}
	if filter.ModuleType != nil || filter.RecordID != nil {
		sub := sq.Select("1").From("playbook_run_records prr").
			Where(sq.Expr("prr.playbook_history_id = playbook_history.id"))
		if filter.ModuleType != nil {
			sub = sub.Where(sq.Eq{"prr.module_type": *filter.ModuleType})
		}
		if filter.RecordID != nil {
			sub = sub.Where(sq.Eq{"prr.record_id": *filter.RecordID})
		}
		stmt = stmt.Where(sq.Expr("EXISTS (?)", sub))
	}
	return stmt
}

// GetPlaybookHistoryById implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistoryById(ctx context.Context, playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error) {
	row, err := w.queriesFromContext(ctx).GetPlaybookHistoryById(ctx, toPgUUID(playbookHistoryId))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var result *json.RawMessage
	if len(row.Result) > 0 {
		raw := json.RawMessage(row.Result)
		result = &raw
	}
	return &domain.PlaybookHistoryResponse{
		ID:           fromPgUUID(row.ID),
		PlaybookID:   fromPgUUID(row.PlaybookID),
		PlaybookData: row.PlaybookData,
		Status:       string(row.Status),
		Error:        fromPgText(row.Error),
		Result:       result,
		TriggeredAt:  row.TriggeredAt.Time,
		Edges:        row.Edges,
	}, nil
}

// GetPlaybookById implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error) {
	pgID, err := toPgUUIDFromString(id)
	if err != nil {
		return nil, err
	}

	row, err := w.queriesFromContext(ctx).GetPlaybookById(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	playbook := toDomainPlaybook(row)
	return &playbook, nil
}

// GetPlaybookGraphById implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookGraphById(ctx context.Context, id string) (*domain.PlaybookGraph, error) {
	pgID, err := toPgUUIDFromString(id)
	if err != nil {
		return nil, err
	}

	row, err := w.queriesFromContext(ctx).GetPlaybookGraphById(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var tasks, edges *json.RawMessage
	if len(row.Tasks) > 0 {
		raw := json.RawMessage(row.Tasks)
		tasks = &raw
	}
	if len(row.Edges) > 0 {
		raw := json.RawMessage(row.Edges)
		edges = &raw
	}
	return &domain.PlaybookGraph{
		ID:                fromPgUUID(row.ID),
		Name:              fromPgTextString(row.Name),
		Description:       fromPgText(row.Description),
		TriggerType:       fromNullTriggerType(row.TriggerType),
		TriggerParameters: row.TriggerParameters,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
		Tasks:             tasks,
		Edges:             edges,
	}, nil
}

// CreatePlaybook implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) CreatePlaybook(ctx context.Context, playbook playbooks.PlaybookPayload) (*domain.Playbooks, error) {
	row, err := w.queriesFromContext(ctx).CreatePlaybook(ctx, db.CreatePlaybookParams{
		Name:              toPgTextFromString(playbook.Name),
		Description:       toPgText(playbook.Description),
		TriggerType:       toNullTriggerTypePtr(playbook.TriggerType),
		TriggerParameters: playbook.TriggerParameters,
	})
	if err != nil {
		return nil, err
	}

	created := toDomainPlaybook(row)
	return &created, nil
}

// UpdatePlaybook implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybook(ctx context.Context, id string, playbook playbooks.UpdatePlaybookData) (*domain.Playbooks, error) {
	pgID, err := toPgUUIDFromString(id)
	if err != nil {
		return nil, err
	}

	var triggerParams []byte
	if playbook.TriggerParameters.Set && playbook.TriggerParameters.Value != nil {
		triggerParams = *playbook.TriggerParameters.Value
	}
	row, err := w.queriesFromContext(ctx).UpdatePlaybook(ctx, db.UpdatePlaybookParams{
		NameSet:              playbook.Name.Set,
		Name:                 toPgTextFromNullable(playbook.Name),
		DescriptionSet:       playbook.Description.Set,
		Description:          toPgTextFromNullable(playbook.Description),
		TriggerTypeSet:       playbook.TriggerType.Set,
		TriggerType:          toNullTriggerTypeFromNullable(playbook.TriggerType),
		TriggerParametersSet: playbook.TriggerParameters.Set,
		TriggerParameters:    triggerParams,
		ID:                   pgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	updated := toDomainPlaybook(row)
	return &updated, nil
}

// CreatePlaybookHistory implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) CreatePlaybookHistory(ctx context.Context, id string, edges []domain.ResponseEdges, run playbooks.RunStamp) (*domain.PlaybookHistory, error) {
	pgID, err := toPgUUIDFromString(id)
	if err != nil {
		return nil, err
	}

	modifiedEdges := make([]map[string]any, len(edges))
	for i, edge := range edges {
		modifiedEdges[i] = map[string]any{
			"id":                    edge.ID,
			"destination_id":        edge.DestinationID,
			"source_id":             edge.SourceID,
			"playbook_id":           edge.PlaybookID,
			"destination_task_name": edge.DestinationTaskName,
			"source_task_name":      edge.SourceTaskName,
			"destination_handle":    utils.NullStringToInterface(edge.DestinationHandle),
			"source_handle":         utils.NullStringToInterface(edge.SourceHandle),
		}
	}
	edgesJSON, _ := json.Marshal(modifiedEdges)

	var inputJSON json.RawMessage
	if run.Input != nil {
		if inputJSON, err = json.Marshal(run.Input); err != nil {
			return nil, err
		}
	}

	row, err := w.queriesFromContext(ctx).CreatePlaybookHistory(ctx, db.CreatePlaybookHistoryParams{
		PlaybookID:  pgID,
		Edges:       edgesJSON,
		TriggerType: toNullTriggerTypePtr(run.TriggerType),
		TriggeredBy: toPgUUIDPtr(run.TriggeredBy),
		Input:       inputJSON,
	})
	if err != nil {
		return nil, err
	}

	history := toDomainPlaybookHistory(row)
	return &history, nil
}

// CreatePlaybookRunRecords implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) CreatePlaybookRunRecords(ctx context.Context, historyID uuid.UUID, moduleType string, recordIDs []uuid.UUID) error {
	q := w.queriesFromContext(ctx)
	for _, recordID := range recordIDs {
		if err := q.CreatePlaybookRunRecord(ctx, db.CreatePlaybookRunRecordParams{
			PlaybookHistoryID: toPgUUID(historyID),
			ModuleType:        moduleType,
			RecordID:          toPgUUID(recordID),
		}); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePlaybookHistory implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybookHistory(ctx context.Context, playbookHistoryId string, playbookHistory playbooks.UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
	pgID, err := toPgUUIDFromString(playbookHistoryId)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(playbookHistory.Result)
	if err != nil {
		return nil, err
	}

	row, err := w.queriesFromContext(ctx).UpdatePlaybookHistory(ctx, db.UpdatePlaybookHistoryParams{
		StatusSet: playbookHistory.Status.Set,
		Status:    toNullPlaybookStatus(playbookHistory.Status),
		ErrorSet:  playbookHistory.Error.Set,
		Error:     toPgTextFromNullable(playbookHistory.Error),
		Result:    jsonData,
		ID:        pgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	history := toDomainPlaybookHistory(row)
	return &history, nil
}

// UpdatePlaybookHistoryStatus implements playbooks.PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybookHistoryStatus(ctx context.Context, playbookHistoryId string, status string) (*domain.PlaybookHistory, error) {
	pgID, err := toPgUUIDFromString(playbookHistoryId)
	if err != nil {
		return nil, err
	}

	row, err := w.queriesFromContext(ctx).UpdatePlaybookHistoryStatus(ctx, db.UpdatePlaybookHistoryStatusParams{
		ID:     pgID,
		Status: db.PlaybookStatus(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	history := toDomainPlaybookHistory(row)
	return &history, nil
}
