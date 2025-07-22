package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RateLimiter represents a simple rate limiter
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*ClientLimit
}

// ClientLimit represents the rate limit for a client
type ClientLimit struct {
	Count    int
	LastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*ClientLimit),
	}
}

// RateLimitMiddleware limits the number of requests per minute for a client
func (rl *RateLimiter) RateLimitMiddleware(limit int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP address
		clientIP := r.RemoteAddr
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			clientIP = ip
		}

		// Check rate limit
		rl.mu.Lock()
		client, exists := rl.clients[clientIP]
		now := time.Now()

		if !exists {
			// New client
			rl.clients[clientIP] = &ClientLimit{
				Count:    1,
				LastSeen: now,
			}
		} else {
			// Reset count if a minute has passed
			if now.Sub(client.LastSeen) > time.Minute {
				client.Count = 1
				client.LastSeen = now
			} else {
				// Increment count
				client.Count++
				client.LastSeen = now

				// Check if limit exceeded
				if client.Count > limit {
					rl.mu.Unlock()
					w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
					w.Header().Set("X-RateLimit-Remaining", "0")
					w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Minute).Unix(), 10))
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-rl.clients[clientIP].Count))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Minute).Unix(), 10))

		rl.mu.Unlock()

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// JWTAuthMiddleware authenticates requests using JWT tokens
func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization format, expected 'Bearer {token}'", http.StatusUnauthorized)
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := r.Context()
		ctx = AddClaimsToContext(ctx, claims)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// APIKeyInfo represents the minimal information needed for API key authentication
type APIKeyInfo struct {
	ID             string
	UserID         string
	Status         string
	ExpiresAt      time.Time
	IPRestrictions []string
	Permissions    map[string]map[string]bool
}

// APIKeyAuthMiddleware authenticates requests using API keys
func APIKeyAuthMiddleware(getAPIKey func(string) (*APIKeyInfo, error), next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from header
		apiKeyHeader := r.Header.Get("X-API-Key")
		if apiKeyHeader == "" {
			http.Error(w, "API key required", http.StatusUnauthorized)
			return
		}

		// Get API key from database
		apiKey, err := getAPIKey(apiKeyHeader)
		if err != nil {
			http.Error(w, "Error validating API key", http.StatusInternalServerError)
			return
		}

		// Check if API key exists
		if apiKey == nil {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Check if API key is active
		if apiKey.Status != "active" {
			http.Error(w, "API key is not active", http.StatusUnauthorized)
			return
		}

		// Check if API key is expired
		if time.Now().After(apiKey.ExpiresAt) {
			http.Error(w, "API key has expired", http.StatusUnauthorized)
			return
		}

		// Check IP restrictions if any
		if len(apiKey.IPRestrictions) > 0 {
			// Get client IP address
			clientIP := r.RemoteAddr
			if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
				clientIP = ip
			}

			// Check if client IP is allowed
			allowed := false
			for _, ipRange := range apiKey.IPRestrictions {
				if strings.HasPrefix(clientIP, ipRange) {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "IP address not allowed", http.StatusForbidden)
				return
			}
		}

		// Add API key to request context
		ctx := r.Context()
		ctx = AddAPIKeyToContext(ctx, apiKey)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// RequirePermissionMiddleware checks if the user has the required permission
func RequirePermissionMiddleware(resource string, action string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get claims from context
		claims := GetClaimsFromContext(r.Context())
		if claims == nil {
			// Try to get API key from context
			apiKeyInfo, ok := GetAPIKeyFromContext(r.Context()).(*APIKeyInfo)
			if !ok || apiKeyInfo == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check API key permissions
			resourcePerms, ok := apiKeyInfo.Permissions[resource]
			if !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Check if API key has the required permission
			hasPermission, ok := resourcePerms[action]
			if !ok || !hasPermission {
				http.Error(w, fmt.Sprintf("Permission denied for %s:%s", resource, action), http.StatusForbidden)
				return
			}
		} else {
			// Check JWT token permissions
			var hasPermission bool

			switch resource {
			case "badges":
				switch action {
				case "read":
					hasPermission = claims.Permissions.Badges.Read
				case "write":
					hasPermission = claims.Permissions.Badges.Write
				case "delete":
					hasPermission = claims.Permissions.Badges.Delete
				}
			case "users":
				switch action {
				case "read":
					hasPermission = claims.Permissions.Users.Read
				case "write":
					hasPermission = claims.Permissions.Users.Write
				case "delete":
					hasPermission = claims.Permissions.Users.Delete
				}
			case "api_keys":
				switch action {
				case "read":
					hasPermission = claims.Permissions.APIKeys.Read
				case "write":
					hasPermission = claims.Permissions.APIKeys.Write
				case "delete":
					hasPermission = claims.Permissions.APIKeys.Delete
				}
			}

			if !hasPermission {
				http.Error(w, fmt.Sprintf("Permission denied for %s:%s", resource, action), http.StatusForbidden)
				return
			}
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}