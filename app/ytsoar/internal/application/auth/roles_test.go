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
	"github.com/yuudev14/ytsoar/internal/types"
)

// The columns are TEXT, so the database accepts any string. A typo would be
// stored happily and then grant nothing, forever — this is the only guard.
func TestCreateRoleRejectsUnknownModule(t *testing.T) {
	env := setupTest(t)

	_, err := env.service.CreateRole(context.Background(), uuid.New(), auth.RoleInput{
		Name:        "auditor",
		Permissions: map[string][]string{"playbook": {"read"}}, // singular: a typo
	})

	assert.ErrorIs(t, err, auth.ErrValidation)
	assert.Contains(t, err.Error(), "playbook")
}

func TestCreateRoleRejectsUnknownAction(t *testing.T) {
	env := setupTest(t)

	_, err := env.service.CreateRole(context.Background(), uuid.New(), auth.RoleInput{
		Name:        "auditor",
		Permissions: map[string][]string{domain.ModulePlaybooks: {"summon"}},
	})

	assert.ErrorIs(t, err, auth.ErrValidation)
}

// Validation must run before anything is written, or a bad matrix leaves a
// role behind with no grants.
func TestCreateRoleValidatesBeforeWriting(t *testing.T) {
	env := setupTest(t)
	// No Create expectation: gomock fails if the repository is touched.

	_, err := env.service.CreateRole(context.Background(), uuid.New(), auth.RoleInput{
		Name:        "auditor",
		Permissions: map[string][]string{"nope": {"read"}},
	})
	require.Error(t, err)
}

func TestCreateRoleWritesValidMatrix(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().Create(gomock.Any(), "auditor", gomock.Any()).
		Return(domain.Role{ID: roleID, Name: "auditor"}, nil)
	env.mockRoles.EXPECT().
		ReplacePermissions(gomock.Any(), roleID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, perms domain.PermissionSet) error {
			assert.True(t, perms.Has(domain.ModulePlaybooks, domain.ActionRead))
			assert.False(t, perms.Has(domain.ModulePlaybooks, domain.ActionExecute))
			return nil
		})
	env.mockRoles.EXPECT().GetWithPermissions(gomock.Any(), roleID).
		Return(domain.RoleWithPermissions{Role: domain.Role{ID: roleID}}, nil)

	_, err := env.service.CreateRole(context.Background(), uuid.New(), auth.RoleInput{
		Name:        "auditor",
		Permissions: map[string][]string{domain.ModulePlaybooks: {domain.ActionRead}},
	})
	require.NoError(t, err)
}

// Builtin roles are the seeded admin/analyst/viewer. Editing them out from
// under the seed would break the bootstrap guarantee.
func TestBuiltinRoleCannotBeModified(t *testing.T) {
	roleID := uuid.New()
	builtin := domain.Role{ID: roleID, Name: domain.RoleAdmin, IsBuiltin: true}

	t.Run("delete", func(t *testing.T) {
		env := setupTest(t)
		env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(builtin, nil)

		err := env.service.DeleteRole(context.Background(), uuid.New(), roleID)
		assert.ErrorIs(t, err, auth.ErrBuiltinRole)
	})

	t.Run("permissions", func(t *testing.T) {
		env := setupTest(t)
		env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(builtin, nil)

		_, err := env.service.SetRolePermissions(context.Background(), uuid.New(), roleID,
			map[string][]string{domain.ModulePlaybooks: {domain.ActionRead}})
		assert.ErrorIs(t, err, auth.ErrBuiltinRole)
	})

	t.Run("rename", func(t *testing.T) {
		env := setupTest(t)
		env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(builtin, nil)

		name := "renamed"
		_, err := env.service.UpdateRole(context.Background(), uuid.New(), roleID,
			auth.UpdateRoleInput{Name: types.Nullable[string]{Value: &name, Set: true}})
		assert.ErrorIs(t, err, auth.ErrBuiltinRole)
	})
}

func TestDeleteCustomRoleSucceeds(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).
		Return(domain.Role{ID: roleID, IsBuiltin: false}, nil)
	env.mockRoles.EXPECT().Delete(gomock.Any(), roleID).Return(int64(1), nil)

	require.NoError(t, env.service.DeleteRole(context.Background(), uuid.New(), roleID))
}

// The DELETE query also filters on is_builtin, so zero rows means the role was
// already gone rather than silently succeeding.
func TestDeleteRoleReportsMissingRow(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).
		Return(domain.Role{ID: roleID}, nil)
	env.mockRoles.EXPECT().Delete(gomock.Any(), roleID).Return(int64(0), nil)

	err := env.service.DeleteRole(context.Background(), uuid.New(), roleID)
	assert.ErrorIs(t, err, auth.ErrRoleNotFound)
}

// ReplacePermissions is a delete followed by N inserts. Outside a transaction
// a failure partway leaves the role holding a matrix nobody chose.
func TestSetRolePermissionsIsTransactional(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID}, nil)
	env.mockRoles.EXPECT().ReplacePermissions(gomock.Any(), roleID, gomock.Any()).Return(nil)
	env.mockRoles.EXPECT().GetWithPermissions(gomock.Any(), roleID).
		Return(domain.RoleWithPermissions{}, nil)

	before := *env.txCalls
	_, err := env.service.SetRolePermissions(context.Background(), uuid.New(), roleID,
		map[string][]string{domain.ModulePlaybooks: {domain.ActionRead}})
	require.NoError(t, err)

	assert.Greater(t, *env.txCalls, before, "the matrix rewrite must run inside a transaction")
}

// A rollback must not be reported as success.
func TestSetRolePermissionsPropagatesRollback(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID}, nil)
	env.mockRoles.EXPECT().ReplacePermissions(gomock.Any(), roleID, gomock.Any()).
		Return(assert.AnError)

	_, err := env.service.SetRolePermissions(context.Background(), uuid.New(), roleID,
		map[string][]string{domain.ModulePlaybooks: {domain.ActionRead}})
	assert.Error(t, err)
}

// An empty matrix is legitimate — a role that grants nothing.
func TestSetRolePermissionsAcceptsEmptyMatrix(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID}, nil)
	env.mockRoles.EXPECT().
		ReplacePermissions(gomock.Any(), roleID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, perms domain.PermissionSet) error {
			assert.Empty(t, perms)
			return nil
		})
	env.mockRoles.EXPECT().GetWithPermissions(gomock.Any(), roleID).
		Return(domain.RoleWithPermissions{}, nil)

	_, err := env.service.SetRolePermissions(context.Background(), uuid.New(), roleID, nil)
	require.NoError(t, err)
}
