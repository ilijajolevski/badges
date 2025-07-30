package apikey

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// Handler handles API key management endpoints
type Handler struct {
	DB     *database.DB
	Logger *zap.Logger
}

// NewHandler creates a new API key handler
func NewHandler(db *database.DB, logger *zap.Logger) *Handler {
	return &Handler{
		DB:     db,
		Logger: logger,
	}
}

// generateUniqueID generates a unique ID using crypto/rand
func generateUniqueID() string {
	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// If there's an error, use timestamp as fallback
		return fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	
	// Convert to hex string
	return hex.EncodeToString(randomBytes)
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name           string   `json:"name"`
	ExpiresAt      string   `json:"expires_at,omitempty"`
	IPRestrictions []string `json:"ip_restrictions,omitempty"`
	Permissions    struct {
		Badges struct {
			Read  bool `json:"read"`
			Write bool `json:"write"`
		} `json:"badges"`
	} `json:"permissions"`
}

// APIKeyResponse represents an API key response
type APIKeyResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Key            string    `json:"key,omitempty"` // Only included when creating a new key
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	LastUsed       time.Time `json:"last_used,omitempty"`
	Status         string    `json:"status"`
	IPRestrictions []string  `json:"ip_restrictions,omitempty"`
	Permissions    struct {
		Badges struct {
			Read  bool `json:"read"`
			Write bool `json:"write"`
		} `json:"badges"`
	} `json:"permissions"`
}

// CreateAPIKey handles the creation of a new API key
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate API key
	apiKey, err := auth.GenerateAPIKey()
	if err != nil {
		h.Logger.Error("Failed to generate API key", zap.Error(err))
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Hash API key for storage
	hashedKey, err := auth.HashAPIKey(apiKey)
	if err != nil {
		h.Logger.Error("Failed to hash API key", zap.Error(err))
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Create API key permissions
	permissions := &database.APIKeyPermissions{
		Badges: struct {
			Read  bool `json:"read"`
			Write bool `json:"write"`
		}{
			Read:  req.Permissions.Badges.Read,
			Write: req.Permissions.Badges.Write,
		},
	}

	// Parse expiration date
	expiresAt := time.Now().AddDate(1, 0, 0) // Default: 1 year
	if req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = parsedTime
		}
	}

	// Create API key in database
	dbAPIKey := &database.APIKey{
		APIKeyID:  generateUniqueID(),
		UserID:    claims.UserID,
		APIKey:    hashedKey,
		Name:      req.Name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Status:    "active",
	}

	// Set permissions
	if err := dbAPIKey.SetPermissions(permissions); err != nil {
		h.Logger.Error("Failed to set API key permissions", zap.Error(err))
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Set IP restrictions
	if err := dbAPIKey.SetIPRestrictions(req.IPRestrictions); err != nil {
		h.Logger.Error("Failed to set API key IP restrictions", zap.Error(err))
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Save API key to database
	if err := h.DB.CreateAPIKey(dbAPIKey); err != nil {
		h.Logger.Error("Failed to create API key in database", zap.Error(err))
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Create response
	resp := APIKeyResponse{
		ID:        dbAPIKey.APIKeyID,
		Name:      dbAPIKey.Name,
		Key:       apiKey, // Include the raw key in the response
		CreatedAt: dbAPIKey.CreatedAt,
		ExpiresAt: dbAPIKey.ExpiresAt,
		Status:    dbAPIKey.Status,
	}

	// Get IP restrictions
	ipRestrictions, err := dbAPIKey.GetIPRestrictions()
	if err == nil {
		resp.IPRestrictions = ipRestrictions
	}

	// Get permissions
	dbPermissions, err := dbAPIKey.GetPermissions()
	if err == nil {
		resp.Permissions.Badges.Read = dbPermissions.Badges.Read
		resp.Permissions.Badges.Write = dbPermissions.Badges.Write
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// ListAPIKeys handles listing all API keys for a user
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get API keys from database
	apiKeys, err := h.DB.ListAPIKeysByUser(claims.UserID)
	if err != nil {
		h.Logger.Error("Failed to list API keys", zap.Error(err))
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	// Create response
	var resp []APIKeyResponse
	for _, apiKey := range apiKeys {
		// Skip if API key is not active
		if apiKey.Status != "active" {
			continue
		}

		// Create response item
		item := APIKeyResponse{
			ID:        apiKey.APIKeyID,
			Name:      apiKey.Name,
			CreatedAt: apiKey.CreatedAt,
			ExpiresAt: apiKey.ExpiresAt,
			Status:    apiKey.Status,
		}

		// Add last used if available
		if apiKey.LastUsed.Valid {
			item.LastUsed = apiKey.LastUsed.Time
		}

		// Get IP restrictions
		ipRestrictions, err := apiKey.GetIPRestrictions()
		if err == nil {
			item.IPRestrictions = ipRestrictions
		}

		// Get permissions
		permissions, err := apiKey.GetPermissions()
		if err == nil {
			item.Permissions.Badges.Read = permissions.Badges.Read
			item.Permissions.Badges.Write = permissions.Badges.Write
		}

		resp = append(resp, item)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RevokeAPIKey handles revoking an API key
func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get API key ID from URL
	apiKeyID := r.URL.Query().Get("id")
	if apiKeyID == "" {
		http.Error(w, "API key ID is required", http.StatusBadRequest)
		return
	}

	// Get API key from database
	apiKey, err := h.DB.GetAPIKey(apiKeyID)
	if err != nil {
		h.Logger.Error("Failed to get API key", zap.Error(err))
		http.Error(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	// Check if API key exists
	if apiKey == nil {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	// Check if API key belongs to user
	if apiKey.UserID != claims.UserID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Update API key status
	apiKey.Status = "revoked"
	if err := h.DB.UpdateAPIKey(apiKey); err != nil {
		h.Logger.Error("Failed to update API key", zap.Error(err))
		http.Error(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// UpdateAPIKey handles updating an API key
func (h *Handler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get API key ID from URL
	apiKeyID := r.URL.Query().Get("id")
	if apiKeyID == "" {
		http.Error(w, "API key ID is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get API key from database
	apiKey, err := h.DB.GetAPIKey(apiKeyID)
	if err != nil {
		h.Logger.Error("Failed to get API key", zap.Error(err))
		http.Error(w, "Failed to update API key", http.StatusInternalServerError)
		return
	}

	// Check if API key exists
	if apiKey == nil {
		http.Error(w, "API key not found", http.StatusNotFound)
		return
	}

	// Check if API key belongs to user
	if apiKey.UserID != claims.UserID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Update API key
	if req.Name != "" {
		apiKey.Name = req.Name
	}

	// Update expiration date
	if req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			apiKey.ExpiresAt = parsedTime
		}
	}

	// Update permissions
	permissions := &database.APIKeyPermissions{
		Badges: struct {
			Read  bool `json:"read"`
			Write bool `json:"write"`
		}{
			Read:  req.Permissions.Badges.Read,
			Write: req.Permissions.Badges.Write,
		},
	}
	if err := apiKey.SetPermissions(permissions); err != nil {
		h.Logger.Error("Failed to set API key permissions", zap.Error(err))
		http.Error(w, "Failed to update API key", http.StatusInternalServerError)
		return
	}

	// Update IP restrictions
	if err := apiKey.SetIPRestrictions(req.IPRestrictions); err != nil {
		h.Logger.Error("Failed to set API key IP restrictions", zap.Error(err))
		http.Error(w, "Failed to update API key", http.StatusInternalServerError)
		return
	}

	// Save API key to database
	if err := h.DB.UpdateAPIKey(apiKey); err != nil {
		h.Logger.Error("Failed to update API key in database", zap.Error(err))
		http.Error(w, "Failed to update API key", http.StatusInternalServerError)
		return
	}

	// Create response
	resp := APIKeyResponse{
		ID:        apiKey.APIKeyID,
		Name:      apiKey.Name,
		CreatedAt: apiKey.CreatedAt,
		ExpiresAt: apiKey.ExpiresAt,
		Status:    apiKey.Status,
	}

	// Add last used if available
	if apiKey.LastUsed.Valid {
		resp.LastUsed = apiKey.LastUsed.Time
	}

	// Get IP restrictions
	ipRestrictions, err := apiKey.GetIPRestrictions()
	if err == nil {
		resp.IPRestrictions = ipRestrictions
	}

	// Get permissions
	dbPermissions, err := apiKey.GetPermissions()
	if err == nil {
		resp.Permissions.Badges.Read = dbPermissions.Badges.Read
		resp.Permissions.Badges.Write = dbPermissions.Badges.Write
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}