package token_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/pkg/token"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	token, gotErr := token.GenerateToken(jwt.MapClaims{})

	if gotErr != nil {
		t.Errorf(gotErr.Error())
	}
	if token == "" {
		t.Errorf(gotErr.Error())
	}
}

func TestGeneratePairToken(t *testing.T) {
	accessToken, refreshToken, gotErr := token.GeneratePairToken(jwt.MapClaims{}, time.Now().Add(time.Hour).Unix())

	if gotErr != nil {
		t.Errorf(gotErr.Error())
	}
	assert.NoError(t, gotErr)
	assert.NotNil(t, accessToken)
	assert.NotNil(t, refreshToken)

}
