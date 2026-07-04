package repository

import (
	"github.com/yuudev14/ytsoar/internal/domain"

	sq "github.com/Masterminds/squirrel"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/logging"
)

type EdgeRepositoryImpl struct {
	*sqlx.DB
}

func NewEdgeRepositoryImpl(db *sqlx.DB) *EdgeRepositoryImpl {
	return &EdgeRepositoryImpl{
		DB: db,
	}
}

// GetNodesByPlaybookId implements EdgeRepository.
func (e *EdgeRepositoryImpl) GetEdgesByPlaybookId(workflowId string) ([]domain.ResponseEdges, error) {
	statement := sq.
		Select("e.*, t1.name AS source_task_name, t2.name AS destination_task_name").
		From("edges e").Join("tasks t1 ON e.source_id = t1.id").
		Join("tasks t2 ON e.destination_id = t2.id").
		Where(sq.Eq{"t1.workflow_id": workflowId})

	return DbExecAndReturnMany[domain.ResponseEdges](
		e.DB,
		statement,
	)
}

// accepts multiple edge structs to be added in the database in a transaction matter
// Do nothing if there's already existing source and destination combined
func (e *EdgeRepositoryImpl) InsertEdges(tx *sqlx.Tx, edges []domain.Edges) ([]domain.Edges, error) {
	statement := sq.Insert("edges").Columns("destination_id", "source_id", "workflow_id", "source_handle", "destination_handle")

	for _, val := range edges {
		statement = statement.Values(val.DestinationID, val.SourceID, val.PlaybookID, val.SourceHandle, val.DestinationHandle)
	}

	statement = statement.Suffix(`ON CONFLICT (destination_id, source_id) DO UPDATE
   	SET source_handle = EXCLUDED.source_handle,
       destination_handle = EXCLUDED.destination_handle RETURNING *`)

	return DbExecAndReturnMany[domain.Edges](
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
func (e *EdgeRepositoryImpl) DeleteAllPlaybookEdges(tx *sqlx.Tx, workflowId string) error {

	// Main delete query with the subquery in both conditions (destination_id and source_id)
	deleteQuery := sq.Delete("edges").Where(`
		destination_id IN  (SELECT id FROM tasks WHERE workflow_id = ?) OR 
		source_id IN (SELECT id FROM tasks WHERE workflow_id = ?)`, workflowId, workflowId)
	// Convert the query to SQL
	sql, args, err := deleteQuery.ToSql()
	logging.Sugar.Debug("DeleteAllPlaybookEdges SQL: ", sql)
	logging.Sugar.Debug("DeleteAllPlaybookEdges Args: ", args)
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
