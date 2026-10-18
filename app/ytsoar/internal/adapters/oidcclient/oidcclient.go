// Package oidcclient is a thin go-oidc/oauth2 wrapper behind auth.OIDCClient. It has no
// policy, only discovery, PKCE params and id_token verification. Provisioning and role
// logic stay in the service, so they stay mockable.
package oidcclient

import (
	"context"
	"errors"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/yuudev14/ytsoar/internal/application/auth"
)

type Client struct{}

func New() *Client { return &Client{} }

// providerAndConfig runs discovery and builds the oauth2 config. Split-horizon dev:
// discover at internal_issuer but trust the public issuer, so the `iss` of the id_token
// (the public url) still verifies.
func (c *Client) providerAndConfig(ctx context.Context, cfg auth.OIDCConfig, redirectURL string) (*oidc.Provider, oauth2.Config, error) {
	discoveryCtx := ctx
	issuer := cfg.Issuer
	if cfg.InternalIssuer != "" {
		discoveryCtx = oidc.InsecureIssuerURLContext(ctx, cfg.Issuer)
		issuer = cfg.InternalIssuer
	}

	provider, err := oidc.NewProvider(discoveryCtx, issuer)
	if err != nil {
		return nil, oauth2.Config{}, err
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	return provider, oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}, nil
}

func (c *Client) AuthCodeURL(ctx context.Context, cfg auth.OIDCConfig, redirectURL, state, codeChallenge string) (string, error) {
	_, oauthCfg, err := c.providerAndConfig(ctx, cfg, redirectURL)
	if err != nil {
		return "", err
	}
	return oauthCfg.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}

func (c *Client) Exchange(ctx context.Context, cfg auth.OIDCConfig, redirectURL, code, codeVerifier string) (auth.OIDCIdentity, error) {
	provider, oauthCfg, err := c.providerAndConfig(ctx, cfg, redirectURL)
	if err != nil {
		return auth.OIDCIdentity{}, err
	}

	oauthToken, err := oauthCfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return auth.OIDCIdentity{}, err
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return auth.OIDCIdentity{}, errors.New("token response carried no id_token")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return auth.OIDCIdentity{}, err
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return auth.OIDCIdentity{}, err
	}

	return auth.OIDCIdentity{
		Subject:           idToken.Subject,
		Email:             stringFromClaim(claims, claimNameOr(cfg.EmailClaim, "email")),
		PreferredUsername: stringFromClaim(claims, claimNameOr(cfg.UsernameClaim, "preferred_username")),
		FirstName:         stringFromClaim(claims, claimNameOr(cfg.FirstNameClaim, "given_name")),
		LastName:          stringFromClaim(claims, claimNameOr(cfg.LastNameClaim, "family_name")),
		Groups:            stringsFromClaim(claims, claimNameOr(cfg.GroupsClaim, "groups")),
		EmailVerified:     claims["email_verified"] == true,
	}, nil
}

func claimNameOr(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}

// claimByPath gets a claim by dotted path, so a nested claim like "realm_access.roles"
// (Keycloak realm roles) works. One segment is a normal top-level lookup.
func claimByPath(claims map[string]any, path string) any {
	var cur any = claims
	for part := range strings.SplitSeq(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	return cur
}

func stringFromClaim(claims map[string]any, path string) string {
	s, _ := claimByPath(claims, path).(string)
	return s
}

// stringsFromClaim reads a claim (maybe a dotted path) that can be a JSON array of
// strings. A single string counts as a one-item list.
func stringsFromClaim(claims map[string]any, path string) []string {
	switch v := claimByPath(claims, path).(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	case string:
		return []string{v}
	default:
		return nil
	}
}
