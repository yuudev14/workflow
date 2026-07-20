package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/types"
)

func TestCreateUserStoresHashNeverPlaintext(t *testing.T) {
	env := setupTest(t)
	actorID := uuid.New()
	roleID := uuid.New()
	newID := uuid.New()

	env.mockHasher.EXPECT().Hash("s3cret-password").Return("argon2-hash", nil)

	env.mockUsers.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params auth.CreateUserParams) (domain.User, error) {
			require.NotNil(t, params.PasswordHash)
			assert.Equal(t, "argon2-hash", *params.PasswordHash)
			assert.NotEqual(t, "s3cret-password", *params.PasswordHash,
				"the plaintext password must never reach the repository")
			assert.Equal(t, domain.AuthProviderLocal, params.AuthProvider)
			return domain.User{ID: newID, Username: params.Username}, nil
		})

	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID}, nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), newID, roleID).Return(nil)
	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), newID).
		Return(domain.UserWithRoles{User: domain.User{ID: newID}, Roles: []string{"analyst"}}, nil)

	created, err := env.service.CreateUser(context.Background(), actorID, auth.CreateUserInput{
		Username: "newbie",
		Email:    "newbie@example.com",
		Password: "s3cret-password",
		RoleIDs:  []string{roleID.String()},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"analyst"}, created.Roles)
}

// An unknown role id must be caught before the insert, otherwise it surfaces
// as an opaque foreign-key violation.
func TestCreateUserRejectsUnknownRole(t *testing.T) {
	env := setupTest(t)
	roleID := uuid.New()
	newID := uuid.New()

	env.mockHasher.EXPECT().Hash(gomock.Any()).Return("hash", nil)
	env.mockUsers.EXPECT().Create(gomock.Any(), gomock.Any()).
		Return(domain.User{ID: newID}, nil)
	env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).
		Return(domain.Role{}, auth.ErrRoleNotFound)

	_, err := env.service.CreateUser(context.Background(), uuid.New(), auth.CreateUserInput{
		Username: "newbie",
		Email:    "newbie@example.com",
		Password: "s3cret-password",
		RoleIDs:  []string{roleID.String()},
	})

	assert.ErrorIs(t, err, auth.ErrRoleNotFound)
}

// A malformed role id must fail before anything is hashed or written.
func TestCreateUserRejectsMalformedRoleID(t *testing.T) {
	env := setupTest(t)

	_, err := env.service.CreateUser(context.Background(), uuid.New(), auth.CreateUserInput{
		Username: "newbie",
		Email:    "newbie@example.com",
		Password: "s3cret-password",
		RoleIDs:  []string{"not-a-uuid"},
	})

	assert.ErrorIs(t, err, auth.ErrValidation)
}

// The security-relevant half of deactivation: flipping is_active alone would
// leave the user working until their refresh token expires — up to a week.
func TestDeactivateUserRevokesEverySession(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()

	env.mockUsers.EXPECT().
		Update(gomock.Any(), userID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, params auth.UpdateUserParams) (domain.User, error) {
			require.True(t, params.IsActive.Set)
			require.NotNil(t, params.IsActive.Value)
			assert.False(t, *params.IsActive.Value)
			return domain.User{ID: userID, IsActive: false}, nil
		})
	env.mockTokens.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)
	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), userID).Return(domain.UserWithRoles{}, nil)

	require.NoError(t, env.service.DeactivateUser(context.Background(), uuid.New(), userID))
}

// The mirror of the test above: an ordinary edit must not log the user out.
func TestUpdateUserWithoutDeactivationKeepsSessions(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	newEmail := "moved@example.com"

	env.mockUsers.EXPECT().Update(gomock.Any(), userID, gomock.Any()).
		Return(domain.User{ID: userID}, nil)
	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), userID).Return(domain.UserWithRoles{}, nil)
	// No RevokeAllForUser expectation: gomock fails the test if it is called.

	_, err := env.service.UpdateUser(context.Background(), uuid.New(), userID, auth.UpdateUserInput{
		Email: types.Nullable[string]{Value: &newEmail, Set: true},
	})
	require.NoError(t, err)
}

// Reactivating is not a deactivation, so it must not revoke either.
func TestUpdateUserReactivatingKeepsSessions(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	active := true

	env.mockUsers.EXPECT().Update(gomock.Any(), userID, gomock.Any()).
		Return(domain.User{ID: userID, IsActive: true}, nil)
	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), userID).Return(domain.UserWithRoles{}, nil)

	_, err := env.service.UpdateUser(context.Background(), uuid.New(), userID, auth.UpdateUserInput{
		IsActive: types.Nullable[bool]{Value: &active, Set: true},
	})
	require.NoError(t, err)
}

// An admin resetting a password is usually responding to a compromise; the
// old sessions have to die with the old password.
func TestSetUserPasswordRevokesEverySession(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()

	env.mockHasher.EXPECT().Hash("brand-new-password").Return("new-hash", nil)
	env.mockUsers.EXPECT().SetPassword(gomock.Any(), userID, "new-hash").Return(nil)
	env.mockTokens.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)

	require.NoError(t, env.service.SetUserPassword(context.Background(), uuid.New(), userID, "brand-new-password"))
}

// Replacing roles must clear the old grants first — appending would silently
// widen access instead of setting it.
func TestSetUserRolesReplacesRatherThanAppends(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	roleID := uuid.New()

	gomock.InOrder(
		env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), userID).Return(nil),
		env.mockRoles.EXPECT().GetByID(gomock.Any(), roleID).Return(domain.Role{ID: roleID}, nil),
		env.mockRoles.EXPECT().AssignToUser(gomock.Any(), userID, roleID).Return(nil),
	)
	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), userID).
		Return(domain.UserWithRoles{Roles: []string{"viewer"}}, nil)

	result, err := env.service.SetUserRoles(context.Background(), uuid.New(), userID, []string{roleID.String()})
	require.NoError(t, err)
	assert.Equal(t, []string{"viewer"}, result.Roles)
}

func TestListUsersReturnsPageAndTotal(t *testing.T) {
	env := setupTest(t)
	filter := auth.UserFilter{}

	env.mockUsers.EXPECT().List(gomock.Any(), 0, 10, filter).
		Return([]domain.UserWithRoles{{User: domain.User{Username: "a"}}}, nil)
	env.mockUsers.EXPECT().Count(gomock.Any(), filter).Return(42, nil)

	page, err := env.service.ListUsers(context.Background(), 0, 10, filter)
	require.NoError(t, err)
	assert.Len(t, page.Entries, 1)
	assert.Equal(t, 42, page.Total, "the total must come from Count, not the page length")
}

func TestGetUserPropagatesNotFound(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()

	env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), userID).
		Return(domain.UserWithRoles{}, auth.ErrUserNotFound)

	_, err := env.service.GetUser(context.Background(), userID)
	assert.True(t, errors.Is(err, auth.ErrUserNotFound))
}
