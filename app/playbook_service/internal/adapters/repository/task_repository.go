package repository

import (
	"context"
	"encoding/json"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
)

type TaskRepositoryImpl struct {
	q    QuerierTx
	pool *pgxpool.Pool
}

func NewTaskRepositoryImpl(q QuerierTx, pool *pgxpool.Pool) *TaskRepositoryImpl {
	return &TaskRepositoryImpl{q: q, pool: pool}
}

func (t *TaskRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return t.q.WithTx(tx)
	}
	return t.q
}

func toDomainTask(row db.Task) domain.Tasks {
	return domain.Tasks{
		ID:            fromPgUUID(row.ID),
		PlaybookID:    fromPgUUID(row.PlaybookID).String(),
		Name:          fromPgTextString(row.Name),
		Config:        fromPgText(row.Config),
		ConnectorName: fromPgText(row.ConnectorName),
		ConnectorID:   fromPgText(row.ConnectorID),
		Operation:     fromPgTextString(row.Operation),
		Description:   fromPgTextString(row.Description),
		Parameters:    row.Parameters,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		X:             float32(row.X.Float64),
		Y:             float32(row.Y.Float64),
	}
}

func toDomainTaskHistory(row db.TaskHistory) domain.TaskHistory {
	var result *json.RawMessage
	if len(row.Result) > 0 {
		raw := json.RawMessage(row.Result)
		result = &raw
	}

	destinationIDs := make([]uuid.UUID, 0, len(row.DestinationIds))
	for _, id := range row.DestinationIds {
		destinationIDs = append(destinationIDs, fromPgUUID(id))
	}

	return domain.TaskHistory{
		ID:                fromPgUUID(row.ID),
		PlaybookHistoryID: fromPgUUID(row.PlaybookHistoryID),
		TaskID:            fromPgUUID(row.TaskID),
		Status:            string(row.Status),
		Error:             fromPgText(row.Error),
		Result:            result,
		TriggeredAt:       row.TriggeredAt.Time,
		Name:              fromPgTextString(row.Name),
		Config:            fromPgText(row.Config),
		ConnectorName:     fromPgText(row.ConnectorName),
		ConnectorID:       fromPgText(row.ConnectorID),
		Operation:         fromPgTextString(row.Operation),
		Description:       fromPgTextString(row.Description),
		Parameters:        row.Parameters,
		X:                 float32(row.X.Float64),
		Y:                 float32(row.Y.Float64),
		DestinationIDs:    destinationIDs,
	}
}

// GetTasksByPlaybookId implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) GetTasksByPlaybookId(ctx context.Context, playbookId string) []domain.Tasks {
	pgID, err := toPgUUIDFromString(playbookId)
	if err != nil {
		logging.Sugar.Warn(err)
		return nil
	}

	rows, err := t.queriesFromContext(ctx).GetTasksByPlaybookId(ctx, pgID)
	if err != nil {
		logging.Sugar.Warn(err)
		return nil
	}

	result := make([]domain.Tasks, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainTask(row))
	}
	return result
}

// UpsertTasks inserts the tasks or updates their content when they already
// exist for the playbook.
func (t *TaskRepositoryImpl) UpsertTasks(ctx context.Context, playbookId uuid.UUID, taskList []domain.Tasks) ([]domain.Tasks, error) {
	q := t.queriesFromContext(ctx)

	upserted := make([]domain.Tasks, 0, len(taskList))
	for _, val := range taskList {
		row, err := q.UpsertTask(ctx, db.UpsertTaskParams{
			PlaybookID:    toPgUUID(playbookId),
			Name:          toPgTextFromString(val.Name),
			Description:   toPgTextFromString(val.Description),
			Parameters:    val.Parameters,
			Config:        toPgText(val.Config),
			ConnectorName: toPgText(val.ConnectorName),
			ConnectorID:   toPgText(val.ConnectorID),
			Operation:     toPgTextFromString(val.Operation),
			X:             toPgFloat8(val.X),
			Y:             toPgFloat8(val.Y),
		})
		if err != nil {
			return nil, err
		}
		upserted = append(upserted, toDomainTask(row))
	}
	return upserted, nil
}

// DeleteTasks implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) DeleteTasks(ctx context.Context, taskIds []uuid.UUID) error {
	ids := make([]pgtype.UUID, 0, len(taskIds))
	for _, id := range taskIds {
		ids = append(ids, toPgUUID(id))
	}
	return t.queriesFromContext(ctx).DeleteTasks(ctx, ids)
}

// CreateTaskHistory implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) CreateTaskHistory(ctx context.Context, playbookHistoryId string, taskList []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error) {
	pgHistoryID, err := toPgUUIDFromString(playbookHistoryId)
	if err != nil {
		return nil, err
	}

	q := t.queriesFromContext(ctx)

	histories := make([]domain.TaskHistory, 0, len(taskList))
	for _, val := range taskList {
		parameters, err := json.Marshal(val.Parameters)
		if err != nil {
			return nil, err
		}

		destinationIds := graph[val.ID]
		pgDestinationIds := make([]pgtype.UUID, 0, len(destinationIds))
		for _, id := range destinationIds {
			pgDestinationIds = append(pgDestinationIds, toPgUUID(id))
		}

		row, err := q.CreateTaskHistory(ctx, db.CreateTaskHistoryParams{
			PlaybookHistoryID: pgHistoryID,
			TaskID:            toPgUUID(val.ID),
			Name:              toPgTextFromString(val.Name),
			Description:       toPgTextFromString(val.Description),
			Parameters:        parameters,
			Config:            toPgText(val.Config),
			X:                 toPgFloat8(val.X),
			Y:                 toPgFloat8(val.Y),
			ConnectorName:     toPgText(val.ConnectorName),
			ConnectorID:       toPgText(val.ConnectorID),
			Operation:         toPgTextFromString(val.Operation),
			DestinationIds:    pgDestinationIds,
		})
		if err != nil {
			return nil, err
		}
		histories = append(histories, toDomainTaskHistory(row))
	}
	return histories, nil
}

// UpdateTaskStatus implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) UpdateTaskStatus(ctx context.Context, playbookHistoryId string, taskId string, status string) (*domain.TaskHistory, error) {
	pgHistoryID, err := toPgUUIDFromString(playbookHistoryId)
	if err != nil {
		return nil, err
	}
	pgTaskID, err := toPgUUIDFromString(taskId)
	if err != nil {
		return nil, err
	}

	row, err := t.queriesFromContext(ctx).UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		PlaybookHistoryID: pgHistoryID,
		TaskID:            pgTaskID,
		Status:            db.TaskStatus(status),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	history := toDomainTaskHistory(row)
	return &history, nil
}

// UpdateTaskHistory implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) UpdateTaskHistory(ctx context.Context, playbookHistoryId string, taskId string, taskHistory tasks.UpdateTaskHistoryData) (*domain.TaskHistory, error) {
	pgHistoryID, err := toPgUUIDFromString(playbookHistoryId)
	if err != nil {
		return nil, err
	}
	pgTaskID, err := toPgUUIDFromString(taskId)
	if err != nil {
		return nil, err
	}

	result, err := json.Marshal(taskHistory.Result)
	if err != nil {
		return nil, err
	}

	parameters, err := json.Marshal(taskHistory.Parameters)
	if err != nil {
		return nil, err
	}

	row, err := t.queriesFromContext(ctx).UpdateTaskHistory(ctx, db.UpdateTaskHistoryParams{
		Name:              toPgTextFromString(taskHistory.Name),
		Description:       toPgTextFromString(taskHistory.Description),
		Parameters:        parameters,
		Result:            result,
		X:                 toPgFloat8(taskHistory.X),
		Y:                 toPgFloat8(taskHistory.Y),
		Operation:         toPgTextFromString(taskHistory.Operation),
		StatusSet:         taskHistory.Status.Set,
		Status:            toNullTaskStatus(taskHistory.Status),
		ErrorSet:          taskHistory.Error.Set,
		Error:             toPgTextFromNullable(taskHistory.Error),
		ConnectorNameSet:  taskHistory.ConnectorName.Set,
		ConnectorName:     toPgTextFromNullable(taskHistory.ConnectorName),
		ConnectorIDSet:    taskHistory.ConnectorID.Set,
		ConnectorID:       toPgTextFromNullable(taskHistory.ConnectorID),
		ConfigSet:         taskHistory.Config.Set,
		Config:            toPgTextFromNullable(taskHistory.Config),
		PlaybookHistoryID: pgHistoryID,
		TaskID:            pgTaskID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	history := toDomainTaskHistory(row)
	return &history, nil
}

// GetTaskHistoryByPlaybookHistoryId implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) GetTaskHistoryByPlaybookHistoryId(ctx context.Context, id string, filter tasks.TaskHistoryFilter) ([]domain.TaskHistory, error) {
	pgID, err := toPgUUIDFromString(id)
	if err != nil {
		return nil, err
	}

	rows, err := t.queriesFromContext(ctx).GetTaskHistoryByPlaybookHistoryId(ctx, pgID)
	if err != nil {
		return nil, err
	}

	histories := make([]domain.TaskHistory, 0, len(rows))
	for _, row := range rows {
		histories = append(histories, toDomainTaskHistory(row))
	}
	return histories, nil
}

// GetTaskHistoryCount implements tasks.TaskRepository.
func (t *TaskRepositoryImpl) GetTaskHistoryCount(ctx context.Context, filter tasks.TaskHistoryFilter) (int, error) {
	stmt := sq.Select("count(*)").From("task_history").
		PlaceholderFormat(sq.Dollar)

	if filter.PlaybookID != nil {
		stmt = stmt.Where(sq.Eq{"playbook_history_id": *filter.PlaybookID})
	}

	return CollectOneScalarFromSqlizer[int](ctx, stmt, t.pool)
}
