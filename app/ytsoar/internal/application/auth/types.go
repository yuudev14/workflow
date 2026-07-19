package auth

import (
	"errors"
	"time"

	"github.com/yuudev14/ytsoar/internal/domain"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrRoleNotFound       = errors.New("role not found")
	ErrTokenNotFound      = errors.New("refresh token not found")
)

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash *string
	FirstName    *string
	LastName     *string
	AuthProvider domain.AuthProvider
	ExternalID   *string
}

type TokenPair struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
	RefreshExpires  time.Time
}

type Me struct {
	User        domain.User         `json:"user"`
	Roles       []string            `json:"roles"`
	Permissions map[string][]string `json:"permissions"`
}

type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AdminUsername   string
	AdminEmail      string
	AdminPassword   string
}
