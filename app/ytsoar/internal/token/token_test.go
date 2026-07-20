package token_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/internal/token"

	"github.com/golang-jwt/jwt/v5"
)

const parseSecret = "mysecret"

func TestParseRoundTrip(t *testing.T) {
	signed, err := token.GenerateToken(jwt.MapClaims{
		"sub": "user-1",
		"typ": "access",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, parseSecret)
	require.NoError(t, err)

	claims, err := token.Parse(signed, parseSecret)

	require.NoError(t, err)
	assert.Equal(t, "user-1", claims["sub"])
	assert.Equal(t, "access", claims["typ"])
}

func TestParseRejectsExpiredToken(t *testing.T) {
	signed, err := token.GenerateToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(-time.Minute).Unix(),
	}, parseSecret)
	require.NoError(t, err)

	_, err = token.Parse(signed, parseSecret)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestParseRejectsWrongSecret(t *testing.T) {
	signed, err := token.GenerateToken(jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, parseSecret)
	require.NoError(t, err)

	_, err = token.Parse(signed, "someone-elses-secret")

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

// The classic JWT forgery: strip the signature and claim the algorithm is
// "none". Parse pins HS256 precisely so this cannot work.
func TestParseRejectsAlgNone(t *testing.T) {
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "attacker",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	forged, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = token.Parse(forged, parseSecret)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

// A token with no exp claim would never expire, so it is refused outright.
func TestParseRequiresExpiry(t *testing.T) {
	signed, err := token.GenerateToken(jwt.MapClaims{"sub": "user-1"}, parseSecret)
	require.NoError(t, err)

	_, err = token.Parse(signed, parseSecret)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestParseRejectsGarbage(t *testing.T) {
	_, err := token.Parse("not-a-jwt", parseSecret)

	assert.ErrorIs(t, err, token.ErrInvalidToken)
}

func TestGenerateToken(t *testing.T) {
	token, gotErr := token.GenerateToken(jwt.MapClaims{}, "mysecret")

	if gotErr != nil {
		t.Errorf("%s", gotErr.Error())
	}
	if token == "" {
		t.Errorf("%s", gotErr.Error())
	}
}

func TestGeneratePairToken(t *testing.T) {
	accessToken, refreshToken, gotErr := token.GeneratePairToken(jwt.MapClaims{}, time.Now().Add(time.Hour).Unix(), "mysecret")

	if gotErr != nil {
		t.Errorf("%s", gotErr.Error())
	}
	assert.NoError(t, gotErr)
	assert.NotNil(t, accessToken)
	assert.NotNil(t, refreshToken)

}
