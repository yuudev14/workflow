package security_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/security"
)

func newKey(t *testing.T) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(rand.Text() + rand.Text())[:32])
}

func TestSecretBox(t *testing.T) {
	box, err := security.NewSecretBox(newKey(t))
	require.NoError(t, err)
	sealed, err := box.Seal("s3cr3t")
	require.NoError(t, err)

	t.Run("opens what it sealed", func(t *testing.T) {
		assert.NotContains(t, sealed, "s3cr3t")
		plain, err := box.Open(sealed)
		require.NoError(t, err)
		assert.Equal(t, "s3cr3t", plain)
	})
	t.Run("fails with another key", func(t *testing.T) {
		other, err := security.NewSecretBox(newKey(t))
		require.NoError(t, err)
		_, err = other.Open(sealed)
		assert.Error(t, err)
	})
	t.Run("fails on a plain value", func(t *testing.T) {
		_, err := box.Open("s3cr3t")
		assert.Error(t, err)
	})
	t.Run("refuses a key that is not 32 bytes", func(t *testing.T) {
		_, err := security.NewSecretBox(base64.StdEncoding.EncodeToString([]byte("short")))
		assert.Error(t, err)
	})
}
