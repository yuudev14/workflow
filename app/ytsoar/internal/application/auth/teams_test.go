package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func TestCreateTeamAssignsMembers(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()
	memberID := uuid.New()

	env.mockTeams.EXPECT().Create(gomock.Any(), "soc-tier1", gomock.Any()).
		Return(domain.Team{ID: teamID, Name: "soc-tier1"}, nil)
	env.mockTeams.EXPECT().
		ReplaceMembers(gomock.Any(), teamID, []uuid.UUID{memberID}).
		Return(nil)
	env.mockTeams.EXPECT().
		ReplaceRoles(gomock.Any(), teamID, []uuid.UUID{}).
		Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{Team: domain.Team{ID: teamID}}, nil)

	_, err := env.service.CreateTeam(context.Background(), uuid.New(), auth.TeamInput{
		Name:      "soc-tier1",
		MemberIDs: []string{memberID.String()},
	})
	require.NoError(t, err)
}

// A malformed id must fail before the team row is written, or a create leaves
// an empty team behind.
func TestCreateTeamRejectsMalformedMemberID(t *testing.T) {
	env := setupTest(t)

	_, err := env.service.CreateTeam(context.Background(), uuid.New(), auth.TeamInput{
		Name:      "soc-tier1",
		MemberIDs: []string{"not-a-uuid"},
	})

	assert.ErrorIs(t, err, auth.ErrValidation)
	assert.Contains(t, err.Error(), "member_ids")
}

func TestSetTeamMembersReplacesRatherThanAppends(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()
	memberID := uuid.New()

	env.mockTeams.EXPECT().
		ReplaceMembers(gomock.Any(), teamID, []uuid.UUID{memberID}).
		Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{}, nil)

	_, err := env.service.SetTeamMembers(context.Background(), uuid.New(), teamID,
		[]string{memberID.String()})
	require.NoError(t, err)
}

func TestSetTeamMembersIsTransactional(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()
	memberID := uuid.New()

	env.mockTeams.EXPECT().ReplaceMembers(gomock.Any(), teamID, gomock.Any()).Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{}, nil)

	before := *env.txCalls
	_, err := env.service.SetTeamMembers(context.Background(), uuid.New(), teamID,
		[]string{memberID.String()})
	require.NoError(t, err)

	assert.Greater(t, *env.txCalls, before, "the member rewrite must run inside a transaction")
}

// Clearing a team is legitimate and must reach the repository as an empty
// list, not be skipped.
func TestSetTeamMembersAcceptsEmptyList(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()

	env.mockTeams.EXPECT().
		ReplaceMembers(gomock.Any(), teamID, []uuid.UUID{}).
		Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{}, nil)

	_, err := env.service.SetTeamMembers(context.Background(), uuid.New(), teamID, nil)
	require.NoError(t, err)
}

func TestDeleteTeamReportsMissingRow(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()

	env.mockTeams.EXPECT().Delete(gomock.Any(), teamID).Return(int64(0), nil)

	err := env.service.DeleteTeam(context.Background(), uuid.New(), teamID)
	assert.ErrorIs(t, err, auth.ErrTeamNotFound)
}

func TestListTeamsReturnsPageAndTotal(t *testing.T) {
	env := setupTest(t)
	filter := auth.TeamFilter{}

	env.mockTeams.EXPECT().List(gomock.Any(), 0, 10, filter).
		Return([]domain.TeamWithMembers{{Team: domain.Team{Name: "soc"}}}, nil)
	env.mockTeams.EXPECT().Count(gomock.Any(), filter).Return(7, nil)

	page, err := env.service.ListTeams(context.Background(), 0, 10, filter)
	require.NoError(t, err)
	assert.Len(t, page.Entries, 1)
	assert.Equal(t, 7, page.Total)
}

func TestSetTeamRolesReplacesWholesaleInTx(t *testing.T) {
	env := setupTest(t)
	teamID, roleID := uuid.New(), uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID, Name: "analyst"}, nil)
	env.mockTeams.EXPECT().ReplaceRoles(gomock.Any(), teamID, []uuid.UUID{roleID}).Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{Team: domain.Team{ID: teamID}}, nil)

	before := *env.txCalls
	_, err := env.service.SetTeamRoles(context.Background(), uuid.New(), teamID, []string{roleID.String()})
	require.NoError(t, err)
	assert.Greater(t, *env.txCalls, before, "delete-then-insert must not half-apply")
}

// An empty list clears the grants rather than no-opping - otherwise a team's
// privileges could never be taken away through the API.
func TestSetTeamRolesEmptyListClears(t *testing.T) {
	env := setupTest(t)
	teamID := uuid.New()

	env.mockTeams.EXPECT().ReplaceRoles(gomock.Any(), teamID, []uuid.UUID{}).Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{Team: domain.Team{ID: teamID}}, nil)

	_, err := env.service.SetTeamRoles(context.Background(), uuid.New(), teamID, []string{})
	require.NoError(t, err)
}

// An unknown role must fail before any write, or it surfaces as an opaque FK
// violation after the delete has already landed.
func TestSetTeamRolesRejectsUnknownRole(t *testing.T) {
	env := setupTest(t)
	teamID, roleID := uuid.New(), uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{}, auth.ErrRoleNotFound)
	// no ReplaceRoles expectation: gomock fails if the write is attempted.

	_, err := env.service.SetTeamRoles(context.Background(), uuid.New(), teamID, []string{roleID.String()})
	assert.ErrorIs(t, err, auth.ErrRoleNotFound)
}

func TestSetTeamRolesRejectsMalformedID(t *testing.T) {
	env := setupTest(t)
	_, err := env.service.SetTeamRoles(context.Background(), uuid.New(), uuid.New(), []string{"not-a-uuid"})
	assert.Error(t, err)
}

// A builtin role may be GRANTED to a team - that only uses the role. Only
// editing a builtin's own matrix is forbidden.
func TestSetTeamRolesAllowsBuiltinRole(t *testing.T) {
	env := setupTest(t)
	teamID, adminRole := uuid.New(), uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), adminRole).
		Return(domain.Role{ID: adminRole, Name: "admin", IsBuiltin: true}, nil)
	env.mockTeams.EXPECT().ReplaceRoles(gomock.Any(), teamID, []uuid.UUID{adminRole}).Return(nil)
	env.mockTeams.EXPECT().GetWithMembers(gomock.Any(), teamID).
		Return(domain.TeamWithMembers{Team: domain.Team{ID: teamID}}, nil)

	_, err := env.service.SetTeamRoles(context.Background(), uuid.New(), teamID, []string{adminRole.String()})
	require.NoError(t, err)
}
