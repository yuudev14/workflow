package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/domain"
)

type EdgeRepositoryImpl struct {
	q    QuerierTx
	pool *pgxpool.Pool
}

func NewEdgeRepositoryImpl(q QuerierTx, pool *pgxpool.Pool) *EdgeRepositoryImpl {
	return &EdgeRepositoryImpl{q: q, pool: pool}
}

func (e *EdgeRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return e.q.WithTx(tx)
	}
	return e.q
}

func toDomainEdge(row db.Edge) domain.Edges {
	return domain.Edges{
		ID:                fromPgUUID(row.ID),
		PlaybookID:        fromPgUUID(row.PlaybookID),
		DestinationID:     fromPgUUID(row.DestinationID),
		SourceID:          fromPgUUID(row.SourceID),
		SourceHandle:      toNullString(row.SourceHandle),
		DestinationHandle: toNullString(row.DestinationHandle),
	}
}

// InsertEdges upserts the given edges; existing source/destination pairs get
// their handles updated.
func (e *EdgeRepositoryImpl) InsertEdges(ctx context.Context, edges []domain.Edges) ([]domain.Edges, error) {
	q := e.queriesFromContext(ctx)

	inserted := make([]domain.Edges, 0, len(edges))
	for _, val := range edges {
		row, err := q.UpsertEdge(ctx, db.UpsertEdgeParams{
			DestinationID:     toPgUUID(val.DestinationID),
			SourceID:          toPgUUID(val.SourceID),
			PlaybookID:        toPgUUID(val.PlaybookID),
			SourceHandle:      toPgTextFromNullString(val.SourceHandle),
			DestinationHandle: toPgTextFromNullString(val.DestinationHandle),
		})
		if err != nil {
			return nil, err
		}
		inserted = append(inserted, toDomainEdge(row))
	}
	return inserted, nil
}

// DeleteEdges implements edges.EdgeRepository.
func (e *EdgeRepositoryImpl) DeleteEdges(ctx context.Context, edgeIds []uuid.UUID) error {
	ids := make([]pgtype.UUID, 0, len(edgeIds))
	for _, id := range edgeIds {
		ids = append(ids, toPgUUID(id))
	}
	return e.queriesFromContext(ctx).DeleteEdges(ctx, ids)
}

// DeleteAllPlaybookEdges implements edges.EdgeRepository.
func (e *EdgeRepositoryImpl) DeleteAllPlaybookEdges(ctx context.Context, playbookId string) error {
	pgID, err := toPgUUIDFromString(playbookId)
	if err != nil {
		return err
	}
	return e.queriesFromContext(ctx).DeleteAllPlaybookEdges(ctx, pgID)
}

// GetEdgesByPlaybookId implements edges.EdgeRepository.
func (e *EdgeRepositoryImpl) GetEdgesByPlaybookId(ctx context.Context, playbookId string) ([]domain.ResponseEdges, error) {
	pgID, err := toPgUUIDFromString(playbookId)
	if err != nil {
		return nil, err
	}

	rows, err := e.queriesFromContext(ctx).GetEdgesByPlaybookId(ctx, pgID)
	if err != nil {
		return nil, err
	}

	edges := make([]domain.ResponseEdges, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, domain.ResponseEdges{
			ID:                  fromPgUUID(row.ID),
			DestinationID:       fromPgUUID(row.DestinationID),
			SourceID:            fromPgUUID(row.SourceID),
			PlaybookID:          fromPgUUID(row.PlaybookID),
			DestinationTaskName: fromPgTextString(row.DestinationTaskName),
			SourceTaskName:      fromPgTextString(row.SourceTaskName),
			DestinationHandle:   toNullString(row.DestinationHandle),
			SourceHandle:        toNullString(row.SourceHandle),
		})
	}
	return edges, nil
}
