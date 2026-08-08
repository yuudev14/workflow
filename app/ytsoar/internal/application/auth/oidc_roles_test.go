package auth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDesiredRoleNamesResolution(t *testing.T) {
	cfg := OIDCConfig{
		GroupRoleMapping: RoleMapping{
			"soc-admins":    {"admin", "analyst"}, // one-to-many
			"soc-analysts":  {"analyst"},
			"ignored-group": {}, // explicit empty → contributes nothing, no passthrough
		},
	}
	// mapping wins + expands one-to-many, unmapped "viewer" passes through, the
	// duplicate "soc-admins" and the overlapping "analyst" dedupe.
	got := desiredRoleNames(cfg, []string{"soc-admins", "soc-analysts", "viewer", "ignored-group", "soc-admins"})
	assert.Equal(t, []string{"admin", "analyst", "viewer"}, got)
}

func TestDesiredRoleNamesPassthroughOnly(t *testing.T) {
	// no mapping table at all - every value is a passthrough candidate.
	got := desiredRoleNames(OIDCConfig{}, []string{"analyst", "analyst"})
	assert.Equal(t, []string{"analyst"}, got)
}

func TestRoleMappingDecodesStringOrList(t *testing.T) {
	raw := json.RawMessage(`{"issuer":"x","client_id":"y","group_role_mapping":{"a":"admin","b":["viewer","analyst"]}}`)
	cfg, err := decodeOIDCConfig(raw)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, cfg.GroupRoleMapping["a"])
	assert.Equal(t, []string{"viewer", "analyst"}, cfg.GroupRoleMapping["b"])
}

func TestRoleMappingRejectsNonStringValue(t *testing.T) {
	raw := json.RawMessage(`{"issuer":"x","client_id":"y","group_role_mapping":{"a":123}}`)
	_, err := decodeOIDCConfig(raw)
	assert.Error(t, err)
}

func TestSyncsRoles(t *testing.T) {
	assert.True(t, OIDCConfig{}.SyncsRoles(), "empty defaults to syncing roles")
	assert.True(t, OIDCConfig{SyncMode: SyncModeRoles}.SyncsRoles())
	assert.True(t, OIDCConfig{SyncMode: SyncModeAll}.SyncsRoles())
	assert.False(t, OIDCConfig{SyncMode: SyncModeAttributes}.SyncsRoles())
	assert.False(t, OIDCConfig{SyncMode: SyncModeOff}.SyncsRoles())
}

func TestDecodeOIDCConfigRequiresOpenidScope(t *testing.T) {
	withScopes := func(scopes string) error {
		_, err := decodeOIDCConfig(json.RawMessage(
			`{"issuer":"x","client_id":"y","scopes":` + scopes + `}`))
		return err
	}
	assert.NoError(t, withScopes(`["openid","email"]`))
	assert.NoError(t, withScopes(`[]`), "unset scopes fall back to the adapter default")
	assert.Error(t, withScopes(`["email","profile"]`), "no openid scope means no id_token")
}

func TestSyncsAttributes(t *testing.T) {
	assert.False(t, OIDCConfig{}.SyncsAttributes(), "empty keeps the profile at JIT-only")
	assert.False(t, OIDCConfig{SyncMode: SyncModeRoles}.SyncsAttributes())
	assert.False(t, OIDCConfig{SyncMode: SyncModeOff}.SyncsAttributes())
	assert.True(t, OIDCConfig{SyncMode: SyncModeAttributes}.SyncsAttributes())
	assert.True(t, OIDCConfig{SyncMode: SyncModeAll}.SyncsAttributes())
}
