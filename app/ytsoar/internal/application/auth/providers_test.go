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

// The whole provider object pasted into the config field (a real mistake we hit)
// must be rejected at create, before any row is written — not stored to fail
// later at login. mockProviders has no Create expectation, so a write fails.
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
		Return(auth.AuthProvider{ID: uuid.New(), Type: "oidc", Name: "Keycloak", Enabled: true, Config: cfg}, nil)

	created, err := env.service.CreateProvider(context.Background(), uuid.New(), auth.ProviderInput{
		Type:   "oidc",
		Name:   "Keycloak",
		Config: cfg,
	})
	require.NoError(t, err)
	assert.Equal(t, "Keycloak", created.Name)
	// secret must come back masked, never the raw value.
	assert.NotContains(t, string(created.Config), "\"s\"")
}
