package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/finki/badges/internal/config"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Provider manages authentication with an OIDC provider
type Provider struct {
	config     *config.AuthConfig
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier
	oauthConfig *oauth2.Config
	logger     *zap.Logger
}

// NewProvider creates a new OIDC provider
func NewProvider(ctx context.Context, cfg *config.AuthConfig, logger *zap.Logger) (*Provider, error) {
	if cfg.AuthType != "oidc" {
		return nil, errors.New("only OIDC authentication is supported")
	}

	if cfg.OIDCClientID == "" || cfg.OIDCClientSecret == "" {
		return nil, errors.New("OIDC client ID and client secret are required")
	}

	// Initialize OIDC provider
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	// Configure OAuth2
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Configure ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDCClientID,
	})

	return &Provider{
		config:      cfg,
		provider:    provider,
		verifier:    verifier,
		oauthConfig: oauthConfig,
		logger:      logger,
	}, nil
}

// GetAuthURL returns the URL to redirect the user to for authentication
func (p *Provider) GetAuthURL(state string) string {
	return p.oauthConfig.AuthCodeURL(state)
}

// Exchange exchanges an authorization code for a token
func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.oauthConfig.Exchange(ctx, code)
}

// VerifyIDToken verifies an ID token and returns the claims
func (p *Provider) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in token response")
	}

	return p.verifier.Verify(ctx, rawIDToken)
}

// GetUserInfo gets user information from the ID token
func (p *Provider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	idToken, err := p.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Subject       string `json:"sub"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if !claims.EmailVerified {
		return nil, errors.New("email not verified")
	}

	return &UserInfo{
		UserID:  claims.Subject,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

// UserInfo contains information about an authenticated user
type UserInfo struct {
	UserID  string
	Email   string
	Name    string
	Picture string
}
