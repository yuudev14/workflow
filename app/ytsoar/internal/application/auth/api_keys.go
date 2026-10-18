package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

// apiKeyTouchInterval limits the last_used_at writes. People read it, so a minute is
// precise enough, and most requests then do no write.
const apiKeyTouchInterval = time.Minute

var (
	ErrAPIKeyNotFound = apperr.New(apperr.NotFound, "api key not found")
	ErrAPIKeyExpiry   = apperr.New(apperr.Invalid, "expires_at must be in the future")
)

type CreateAPIKeyInput struct {
	UserID    string     `json:"user_id" binding:"required,uuid"`
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type CreateAPIKeyParams struct {
	UserID    uuid.UUID
	Name      string
	KeyHash   string
	Prefix    string
	CreatedBy uuid.UUID
	ExpiresAt *time.Time
}

// CreatedAPIKey is the only response that holds the key. It cannot be read again.
type CreatedAPIKey struct {
	domain.APIKey
	Key string `json:"key"`
}

type APIKeyFilter struct {
	UserID *string `form:"user_id" binding:"omitempty,uuid"`
}

func (s *Service) CreateAPIKey(ctx context.Context, actorID uuid.UUID, input CreateAPIKeyInput) (CreatedAPIKey, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return CreatedAPIKey{}, ErrValidation
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(s.now()) {
		return CreatedAPIKey{}, ErrAPIKeyExpiry
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return CreatedAPIKey{}, err
	}

	key := domain.APIKeyPrefix + randomToken()
	created, err := s.apiKeys.Create(ctx, CreateAPIKeyParams{
		UserID:    userID,
		Name:      input.Name,
		KeyHash:   hashToken(key),
		Prefix:    key[:len(domain.APIKeyPrefix)+4],
		CreatedBy: actorID,
		ExpiresAt: input.ExpiresAt,
	})
	if err != nil {
		return CreatedAPIKey{}, err
	}
	created.Username = user.Username

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "api_key_created",
		EntityID: entityID(created.ID),
		Detail:   map[string]any{"name": created.Name, "user_id": userID},
	})
	return CreatedAPIKey{APIKey: created, Key: key}, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, filter APIKeyFilter) ([]domain.APIKey, error) {
	var userID *uuid.UUID
	if filter.UserID != nil {
		id, err := uuid.Parse(*filter.UserID)
		if err != nil {
			return nil, ErrValidation
		}
		userID = &id
	}
	return s.apiKeys.List(ctx, userID)
}

func (s *Service) DeleteAPIKey(ctx context.Context, actorID, id uuid.UUID) error {
	deleted, err := s.apiKeys.Delete(ctx, id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrAPIKeyNotFound
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "api_key_deleted",
		EntityID: entityID(id),
	})
	return nil
}

// VerifyAPIKey checks a key from the Authorization header. Every failure is the same
// ErrInvalidCredentials, so a caller cannot tell a deleted key from a wrong one.
func (s *Service) VerifyAPIKey(ctx context.Context, key string) (domain.AuthUser, error) {
	stored, err := s.apiKeys.GetByHash(ctx, hashToken(key))
	if err != nil {
		return domain.AuthUser{}, ErrInvalidCredentials
	}

	now := s.now()
	if stored.ExpiresAt != nil && !now.Before(*stored.ExpiresAt) {
		return domain.AuthUser{}, ErrInvalidCredentials
	}

	if stored.LastUsedAt == nil || now.Sub(*stored.LastUsedAt) >= apiKeyTouchInterval {
		if err := s.apiKeys.Touch(ctx, stored.ID, now); err != nil {
			s.logger.Warnf("could not update last_used_at for api key %s: %v", stored.ID, err)
		}
	}
	return domain.AuthUser{ID: stored.UserID, Username: stored.Username}, nil
}
