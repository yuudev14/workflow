package token_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/token"

	"github.com/golang-jwt/jwt/v5"
)

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
