package repository

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// TeamRepositoryImpl implements auth.TeamRepository.
type TeamRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewTeamRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *TeamRepositoryImpl {
	return &TeamRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *TeamRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

// teamRow carries the members aggregate as raw JSON so one query returns a
// team and its members together rather than N+1.
type teamRow struct {
	domain.Team
	Members json.RawMessage `db:"members"`
}

const membersAggregate = `COALESCE((
    SELECT jsonb_agg(jsonb_build_object('id', u.id, 'username', u.username, 'email', u.email)
                     ORDER BY u.username)
    FROM team_members tm JOIN users u ON u.id = tm.user_id
    WHERE tm.team_id = t.id
), '[]'::jsonb) AS members`

func selectTeams() sq.SelectBuilder {
	return sq.Select("t.*, " + membersAggregate).From("teams t").PlaceholderFormat(sq.Dollar)
}

func applyTeamFilter(stmt sq.SelectBuilder, filter auth.TeamFilter) sq.SelectBuilder {
	if filter.Search != nil {
		term := fmt.Sprint("%", *filter.Search, "%")
		stmt = stmt.Where(sq.Expr("(t.name ILIKE ? OR t.description ILIKE ?)", term, term))
	}
	return stmt
}

func (r *TeamRepositoryImpl) List(ctx context.Context, offset, limit int, filter auth.TeamFilter) ([]domain.TeamWithMembers, error) {
	stmt := applyTeamFilter(selectTeams(), filter).
		OrderBy("t.name").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	rows, err := CollectRowsFromSqlizer[teamRow](ctx, stmt, r.pool, r.logger)
	if err != nil {
		return nil, err
	}

	teams := make([]domain.TeamWithMembers, 0, len(rows))
	for _, row := range rows {
		team, err := toDomainTeam(row)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

func (r *TeamRepositoryImpl) Count(ctx context.Context, filter auth.TeamFilter) (int, error) {
	stmt := applyTeamFilter(
		sq.Select("count(*)").From("teams t").PlaceholderFormat(sq.Dollar), filter)
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

func (r *TeamRepositoryImpl) GetWithMembers(ctx context.Context, id uuid.UUID) (domain.TeamWithMembers, error) {
	rows, err := CollectRowsFromSqlizer[teamRow](
		ctx, selectTeams().Where(sq.Eq{"t.id": id}), r.pool, r.logger)
	if err != nil {
		return domain.TeamWithMembers{}, err
	}
	if len(rows) == 0 {
		return domain.TeamWithMembers{}, auth.ErrTeamNotFound
	}
	return toDomainTeam(rows[0])
}

func (r *TeamRepositoryImpl) Create(ctx context.Context, name string, description *string) (domain.Team, error) {
	row, err := r.queriesFromContext(ctx).CreateTeam(ctx, db.CreateTeamParams{
		Name:        name,
		Description: toPgText(description),
	})
	if err != nil {
		return domain.Team{}, err
	}
	return toDomainTeamRow(row), nil
}

func (r *TeamRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params auth.UpdateTeamParams) (domain.Team, error) {
	row, err := r.queriesFromContext(ctx).UpdateTeam(ctx, db.UpdateTeamParams{
		ID:             toPgUUID(id),
		NameSet:        params.Name.Set,
		Name:           toPgTextFromNullable(params.Name),
		DescriptionSet: params.Description.Set,
		Description:    toPgTextFromNullable(params.Description),
	})
	if err != nil {
		return domain.Team{}, mapNoRows(err, auth.ErrTeamNotFound)
	}
	return toDomainTeamRow(row), nil
}

func (r *TeamRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queriesFromContext(ctx).DeleteTeam(ctx, toPgUUID(id))
}

func (r *TeamRepositoryImpl) ReplaceMembers(ctx context.Context, teamID uuid.UUID, userIDs []uuid.UUID) error {
	q := r.queriesFromContext(ctx)
	if err := q.DeleteTeamMembers(ctx, toPgUUID(teamID)); err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := q.InsertTeamMember(ctx, db.InsertTeamMemberParams{
			TeamID: toPgUUID(teamID),
			UserID: toPgUUID(userID),
		}); err != nil {
			return err
		}
	}
	return nil
}

func toDomainTeam(row teamRow) (domain.TeamWithMembers, error) {
	members := []domain.TeamMember{}
	if len(row.Members) > 0 {
		if err := json.Unmarshal(row.Members, &members); err != nil {
			return domain.TeamWithMembers{}, err
		}
	}
	return domain.TeamWithMembers{Team: row.Team, Members: members}, nil
}

func toDomainTeamRow(row db.Team) domain.Team {
	return domain.Team{
		ID:          fromPgUUID(row.ID),
		Name:        row.Name,
		Description: fromPgText(row.Description),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
