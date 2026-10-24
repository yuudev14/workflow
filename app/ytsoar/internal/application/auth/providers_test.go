package auth_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/auth"
)

// The whole provider object pasted into the config field (a real mistake we had) must be
// rejected at create, before any row is written, and not stored to fail at login.
// mockProviders has no Create expectation, so a write fails.
func TestCreateProviderRejectsWrappedConfig(t *testing.T) {
	env := setupTest(t)
	wrapped := json.RawMessage(`{"name":"Keycloak","type":"oidc","enabled":true,"config":{"issuer":"x","client_id":"y"}}`)

	_, err := env.service.CreateProvider(context.Background(), uuid.New(), auth.ProviderInput{
		Type:   "oidc",
		Name:   "Keycloak",
		Config: wrapped,
	})
	assert.Error(t, err)
}

func TestCreateProviderValidConfigIsWritten(t *testing.T) {
	env := setupTest(t)
	cfg := json.RawMessage(`{"issuer":"https://idp","client_id":"cid","client_secret":"s"}`)

	env.mockProviders.EXPECT().
		Create(gomock.Any(), gomock.Any(), "Keycloak", gomock.Any(), true).
		DoAndReturn(func(_ context.Context, _ any, _ string, written json.RawMessage, _ bool) (auth.AuthProvider, error) {
			assert.JSONEq(t, `{"issuer":"sealed:https://idp","client_id":"sealed:cid","client_secret":"sealed:s"}`, string(written))
			return auth.AuthProvider{ID: uuid.New(), Type: "oidc", Name: "Keycloak", Enabled: true, Config: written}, nil
		})

	created, err := env.service.CreateProvider(context.Background(), uuid.New(), auth.ProviderInput{
		Type:   "oidc",
		Name:   "Keycloak",
		Config: cfg,
	})
	require.NoError(t, err)
	assert.Equal(t, "Keycloak", created.Name)
	// the admin view opens the values and masks the secret
	assert.JSONEq(t, `{"issuer":"https://idp","client_id":"cid","client_secret":"********"}`, string(created.Config))
}

// A masked secret keeps the stored sealed value, and a new secret is sealed.
func TestUpdateProviderSealsSecrets(t *testing.T) {
	env := setupTest(t)
	id := uuid.New()
	stored := json.RawMessage(`{"issuer":"sealed:https://idp","client_id":"sealed:cid","client_secret":"sealed:old"}`)
	env.mockProviders.EXPECT().GetByID(gomock.Any(), id).
		Return(auth.AuthProvider{ID: id, Type: "oidc", Config: stored}, nil).Times(2)

	for _, tc := range []struct{ name, secret, want string }{
		{"keeps the stored secret when it is masked", "********", "sealed:old"},
		{"seals a new secret", "new", "sealed:new"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env.mockProviders.EXPECT().Update(gomock.Any(), id, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, p auth.UpdateProviderParams) (auth.AuthProvider, error) {
					assert.JSONEq(t, `{"issuer":"sealed:https://idp","client_id":"sealed:cid","client_secret":"`+tc.want+`"}`, string(p.Config))
					return auth.AuthProvider{ID: id, Type: "oidc", Config: p.Config}, nil
				})
			_, err := env.service.UpdateProvider(context.Background(), uuid.New(), id, auth.UpdateProviderInput{
				Config: json.RawMessage(`{"issuer":"https://idp","client_id":"cid","client_secret":"` + tc.secret + `"}`),
			})
			require.NoError(t, err)
		})
	}
}

// The admin view masks bind_password too, so an LDAP update must keep the stored one.
func TestUpdateProviderKeepsMaskedBindPassword(t *testing.T) {
	env := setupTest(t)
	id := uuid.New()
	env.mockProviders.EXPECT().GetByID(gomock.Any(), id).
		Return(auth.AuthProvider{ID: id, Type: "ldap", Config: json.RawMessage(`{"bind_password":"sealed:real"}`)}, nil)
	env.mockProviders.EXPECT().Update(gomock.Any(), id, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, p auth.UpdateProviderParams) (auth.AuthProvider, error) {
			assert.JSONEq(t, `{"bind_password":"sealed:real","host":"sealed:ldap2"}`, string(p.Config))
			return auth.AuthProvider{ID: id, Type: "ldap", Config: p.Config}, nil
		})

	_, err := env.service.UpdateProvider(context.Background(), uuid.New(), id, auth.UpdateProviderInput{
		Config: json.RawMessage(`{"bind_password":"********","host":"ldap2"}`),
	})
	require.NoError(t, err)
}
