package oidcclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringsFromClaimNestedPath(t *testing.T) {
	claims := map[string]any{
		"realm_access": map[string]any{"roles": []any{"admin", "analyst"}},
		"groups":       []any{"soc-admins"},
	}
	// Keycloak realm roles live at a nested path.
	assert.Equal(t, []string{"admin", "analyst"}, stringsFromClaim(claims, "realm_access.roles"))
	assert.Equal(t, []string{"soc-admins"}, stringsFromClaim(claims, "groups"))
	assert.Nil(t, stringsFromClaim(claims, "realm_access.missing"))
	assert.Nil(t, stringsFromClaim(claims, "no.such.path"))
}

func TestStringsFromClaimShapes(t *testing.T) {
	assert.Equal(t, []string{"x"}, stringsFromClaim(map[string]any{"g": "x"}, "g"))
	assert.Equal(t, []string{"a", "b"}, stringsFromClaim(map[string]any{"g": []any{"a", "b"}}, "g"))
	assert.Nil(t, stringsFromClaim(map[string]any{"g": 5}, "g"))
}

func TestStringFromClaim(t *testing.T) {
	claims := map[string]any{"email": "a@corp", "profile": map[string]any{"name": "Al"}}
	assert.Equal(t, "a@corp", stringFromClaim(claims, "email"))
	assert.Equal(t, "Al", stringFromClaim(claims, "profile.name"))
	assert.Equal(t, "", stringFromClaim(claims, "missing"))
}

func TestClaimNameOr(t *testing.T) {
	assert.Equal(t, "groups", claimNameOr("", "groups"))
	assert.Equal(t, "roles", claimNameOr("roles", "groups"))
}
