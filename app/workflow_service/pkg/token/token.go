package token

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/yuudev14-workflow/workflow-service/environment"
)

// function to generate a token
func GenerateToken(claims jwt.MapClaims) (string, error) {
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return newToken.SignedString([]byte(environment.Settings.JWT_SECRET))

}

// function to generate access_token and refresh_token
func GeneratePairToken(claims jwt.MapClaims, refreshTokenExp int64) (*string, *string, error) {

	// generate access token
	accessToken, err := GenerateToken(claims)
	if err != nil {
		return nil, nil, err
	}
	// get subject
	refreshTokenSubject, err := claims.GetSubject()

	if err != nil {
		return &accessToken, nil, err
	}

	// generate refresh token
	refreshToken, err := GenerateToken(jwt.MapClaims{
		"sub": refreshTokenSubject,
		"exp": refreshTokenExp,
	})
	if err != nil {
		return &accessToken, nil, err
	}
	return &accessToken, &refreshToken, nil

}
