package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

const (
	inviteTTL = 72 * time.Hour
	resetTTL  = time.Hour
	// resendInterval stops one address from getting many emails
	resendInterval = time.Minute
)

var (
	// ErrInvalidLink is the same for a wrong, used and expired link, so it tells nothing
	ErrInvalidLink  = apperr.New(apperr.Invalid, "the link is not valid or has expired")
	ErrEmailTooSoon = apperr.New(apperr.Conflict, "an email was sent less than a minute ago")
	ErrCannotInvite = apperr.New(apperr.Invalid, "only an active local user can get an invite")
)

type linkEmail struct {
	ttl     time.Duration
	subject string
	body    string
}

var linkEmails = map[string]linkEmail{
	TokenInvite: {inviteTTL, "Your YTSoar account",
		"An administrator made a YTSoar account for you. Your user name is %s.\n\n" +
			"Open this link to set your password. The link works one time and expires in 72 hours.\n\n%s\n"},
	TokenPasswordReset: {resetTTL, "Reset your YTSoar password",
		"Somebody asked to reset the password of the YTSoar account %s.\n\n" +
			"Open this link to set a new password. The link works one time and expires in 1 hour.\n\n%s\n\n" +
			"If you did not ask for this, do nothing. Your password does not change.\n"},
}

// sendLink makes a one-time link and emails it. Only the hash of the token is stored.
func (s *Service) sendLink(ctx context.Context, user domain.User, purpose string) error {
	email := linkEmails[purpose]
	token := randomToken()
	now := s.now()

	issued, err := s.userTokens.Issue(ctx, IssueTokenParams{
		UserID:      user.ID,
		TokenHash:   hashToken(token),
		Purpose:     purpose,
		ExpiresAt:   now.Add(email.ttl),
		CreatedAt:   now,
		ResendAfter: now.Add(-resendInterval),
	})
	if err != nil {
		return err
	}
	if !issued {
		return ErrEmailTooSoon
	}

	link := strings.TrimRight(s.cfg.FrontendURL, "/") + "/set-password?token=" + token
	if err := s.mailer.Send(ctx, user.Email, email.subject, fmt.Sprintf(email.body, user.Username, link)); err != nil {
		return apperr.Wrap(apperr.Unavailable, "could not send the email", err)
	}
	return nil
}

// InviteUser sends the set-password link again, for example when the first one expired.
func (s *Service) InviteUser(ctx context.Context, actorID, id uuid.UUID) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !user.IsActive || user.AuthProvider != domain.AuthProviderLocal {
		return ErrCannotInvite
	}
	if err := s.sendLink(ctx, user, TokenInvite); err != nil {
		return err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "user_invited",
		EntityID: entityID(id),
	})
	return nil
}

// ForgotPassword always succeeds, so the reply does not show if the email exists. The
// email goes out in the background, so the reply time does not show it either.
func (s *Service) ForgotPassword(ctx context.Context, email string) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			s.logger.Errorf("forgot password: could not find the user: %v", err)
		}
		return
	}
	if !user.IsActive || user.AuthProvider != domain.AuthProviderLocal {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Errorf("forgot password panicked: %v", r)
			}
		}()
		err := s.sendLink(context.WithoutCancel(ctx), user, TokenPasswordReset)
		if err != nil && !errors.Is(err, ErrEmailTooSoon) {
			s.logger.Errorf("could not send the password reset email to user %s: %v", user.ID, err)
		}
	}()
}

// ResetPassword sets the password with an invite or a reset link. It also verifies the
// email, because only the owner of the address can open the link, and it ends all the
// sessions of the user.
func (s *Service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return err
	}

	var token UserToken
	err = s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		token, err = s.userTokens.Take(ctx, hashToken(input.Token))
		if err != nil {
			return err
		}
		// the rollback keeps the expired row, and the daily job deletes it
		if !s.now().Before(token.ExpiresAt) {
			return ErrInvalidLink
		}
		if err := s.users.SetPassword(ctx, token.UserID, hash); err != nil {
			return err
		}
		if err := s.users.MarkEmailVerified(ctx, token.UserID, s.now()); err != nil {
			return err
		}
		return s.tokens.RevokeAllForUser(ctx, token.UserID)
	})
	if err != nil {
		return err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID: &token.UserID,
		Module:  "auth",
		Action:  "password_set_by_link",
		Detail:  map[string]any{"purpose": token.Purpose},
	})
	return nil
}
