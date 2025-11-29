package repository

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
	"github.com/yuudev14-workflow/workflow-service/models"
)

type EdgeRepository interface {
	InsertEdges(tx *sqlx.Tx, edges []models.Edges) ([]models.Edges, error)
	DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error
	DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error
	GetEdgesByWorkflowId(workflowId string) ([]Edges, error)
}

type Edges struct {
	ID                  uuid.UUID      `db:"id" json:"id"`
	DestinationID       uuid.UUID      `db:"destination_id" json:"destination_id"`
	SourceID            uuid.UUID      `db:"source_id" json:"source_id"`
	WorkflowID          uuid.UUID      `db:"workflow_id" json:"workflow_id"`
	DestinationTaskName string         `db:"destination_task_name" json:"destination_task_name"`
	SourceTaskName      string         `db:"source_task_name" json:"source_task_name"`
	DestinationHandle   sql.NullString `db:"destination_handle" json:"destination_handle"`
	SourceHandle        sql.NullString `db:"source_handle" json:"source_handle"`
}

type EdgeRepositoryImpl struct {
	*sqlx.DB
}

func NewEdgeRepositoryImpl(db *sqlx.DB) EdgeRepository {
	return &EdgeRepositoryImpl{
		DB: db,
	}
}

// GetNodesByWorkflowId implements EdgeRepository.
func (e *EdgeRepositoryImpl) GetEdgesByWorkflowId(workflowId string) ([]Edges, error) {
	statement := sq.
		Select("e.*, t1.name AS source_task_name, t2.name AS destination_task_name").
		From("edges e").Join("tasks t1 ON e.source_id = t1.id").
		Join("tasks t2 ON e.destination_id = t2.id").
		Where(sq.Eq{"t1.workflow_id": workflowId})

	return DbExecAndReturnMany[Edges](
		e.DB,
		statement,
	)
}

// accepts multiple edge structs to be added in the database in a transaction matter
// Do nothing if there's already existing source and destination combined
func (e *EdgeRepositoryImpl) InsertEdges(tx *sqlx.Tx, edges []models.Edges) ([]models.Edges, error) {
	statement := sq.Insert("edges").Columns("destination_id", "source_id", "workflow_id", "source_handle", "destination_handle")

	for _, val := range edges {
		statement = statement.Values(val.DestinationID, val.SourceID, val.WorkflowID, val.SourceHandle, val.DestinationHandle)
	}

	statement = statement.Suffix(`ON CONFLICT (destination_id, source_id) DO UPDATE
   	SET source_handle = EXCLUDED.source_handle,
       destination_handle = EXCLUDED.destination_handle RETURNING *`)

	return DbExecAndReturnMany[models.Edges](
		tx,
		statement,
	)
}

// accepts multiple edge ids to be deleted
func (e *EdgeRepositoryImpl) DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error {
	sql, args, err := sq.Delete("edges").Where(sq.Eq{"id": edgeIds}).ToSql()
	logging.Sugar.Debug("DeleteEdges SQL: ", sql)
	logging.Sugar.Debug("DeleteEdges Args: ", args)
	if err != nil {
		logging.Sugar.Error("Failed to build SQL query", err)
		return err
	}
	sql = tx.Rebind(sql)
	_, err = tx.Exec(sql, args...)
	if err != nil {
		logging.Sugar.Warn(err)
	}

	return err
}

// accepts multiple edge ids to be deleted
func (e *EdgeRepositoryImpl) DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error {

	// Main delete query with the subquery in both conditions (destination_id and source_id)
	deleteQuery := sq.Delete("edges").Where(`
		destination_id IN  (SELECT id FROM tasks WHERE workflow_id = ?) OR 
		source_id IN (SELECT id FROM tasks WHERE workflow_id = ?)`, workflowId, workflowId)
	// Convert the query to SQL
	sql, args, err := deleteQuery.ToSql()
	logging.Sugar.Debug("DeleteAllWorkflowEdges SQL: ", sql)
	logging.Sugar.Debug("DeleteAllWorkflowEdges Args: ", args)
	if err != nil {
		logging.Sugar.Error("Failed to build SQL query", err)
		return err
	}
	sql = tx.Rebind(sql)
	_, err = tx.Exec(sql, args...)
	if err != nil {
		logging.Sugar.Warn(err)
	}

	return err
}
