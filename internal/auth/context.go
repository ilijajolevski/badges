package auth

import (
	"context"
)

// Context keys
type contextKey string

const (
	claimsContextKey contextKey = "claims"
	apiKeyContextKey contextKey = "api_key"
)

// AddClaimsToContext adds JWT claims to the request context
func AddClaimsToContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaimsFromContext retrieves JWT claims from the request context
func GetClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// AddAPIKeyToContext adds an API key to the request context
func AddAPIKeyToContext(ctx context.Context, apiKey interface{}) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, apiKey)
}

// GetAPIKeyFromContext retrieves an API key from the request context
func GetAPIKeyFromContext(ctx context.Context) interface{} {
	return ctx.Value(apiKeyContextKey)
}