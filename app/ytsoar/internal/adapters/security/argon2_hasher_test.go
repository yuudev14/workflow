package security_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/security"
)

func TestHashAndVerifyRoundTrip(t *testing.T) {
	hasher := security.NewArgon2Hasher()

	encoded, err := hasher.Hash("correct horse battery staple")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(encoded, "$argon2id$"), "expected a PHC-encoded argon2id hash")
	assert.True(t, hasher.Verify("correct horse battery staple", encoded))
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hasher := security.NewArgon2Hasher()

	encoded, err := hasher.Hash("real-password")
	require.NoError(t, err)

	assert.False(t, hasher.Verify("wrong-password", encoded))
}

// The salt is random, so the same password never hashes to the same string.
func TestHashIsSalted(t *testing.T) {
	hasher := security.NewArgon2Hasher()

	first, err := hasher.Hash("same-password")
	require.NoError(t, err)
	second, err := hasher.Hash("same-password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.True(t, hasher.Verify("same-password", first))
	assert.True(t, hasher.Verify("same-password", second))
}

// A user row with a corrupt or empty hash must fail closed, not error out.
func TestVerifyRejectsMalformedHash(t *testing.T) {
	hasher := security.NewArgon2Hasher()

	assert.False(t, hasher.Verify("anything", ""))
	assert.False(t, hasher.Verify("anything", "not-a-hash"))
	assert.False(t, hasher.Verify("anything", "$argon2id$v=19$garbage"))
}
