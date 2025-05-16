package badge

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// API handles badge management API endpoints
type API struct {
	db            *database.DB
	logger        *zap.Logger
	authMiddleware *auth.Middleware
}

// NewAPI creates a new badge management API
func NewAPI(db *database.DB, logger *zap.Logger, authMiddleware *auth.Middleware) *API {
	return &API{
		db:            db,
		logger:        logger,
		authMiddleware: authMiddleware,
	}
}

// ListBadgesHandler handles the request to list all badges
func (a *API) ListBadgesHandler(w http.ResponseWriter, r *http.Request) {
	// Get all badges from the database
	badges, err := a.db.ListBadges()
	if err != nil {
		a.logger.Error("Failed to list badges", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the badges as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(badges); err != nil {
		a.logger.Error("Failed to encode badges", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetBadgeHandler handles the request to get a badge by commit ID
func (a *API) GetBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the commit ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/badges/")
	commitID := strings.Split(path, "/")[0]

	// Get the badge from the database
	badge, err := a.db.GetBadge(commitID)
	if err != nil {
		a.logger.Error("Failed to get badge", zap.Error(err), zap.String("commitID", commitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if badge == nil {
		http.Error(w, "Badge not found", http.StatusNotFound)
		return
	}

	// Return the badge as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(badge); err != nil {
		a.logger.Error("Failed to encode badge", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// CreateBadgeHandler handles the request to create a new badge
func (a *API) CreateBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var badge database.Badge
	if err := json.NewDecoder(r.Body).Decode(&badge); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the user ID from the session
	session, _ := a.authMiddleware.GetSession(r)
	userID, ok := session.Values[auth.SessionUserID].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has access to the badge type
	hasAccess, err := a.db.UserHasAccessToBadgeType(userID, badge.Type)
	if err != nil {
		a.logger.Error("Failed to check badge access", zap.Error(err), zap.String("userID", userID), zap.String("badgeType", badge.Type))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if the user is a superadmin
	user, err := a.db.GetUserByID(userID)
	if err != nil {
		a.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only allow access if the user is a superadmin or has access to the badge type
	if !user.IsSuperadmin && !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Create the badge
	if err := a.db.CreateBadge(&badge); err != nil {
		a.logger.Error("Failed to create badge", zap.Error(err), zap.String("commitID", badge.CommitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the created badge
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(badge); err != nil {
		a.logger.Error("Failed to encode badge", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// UpdateBadgeHandler handles the request to update a badge
func (a *API) UpdateBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow PUT requests
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the commit ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/badges/")
	commitID := strings.Split(path, "/")[0]

	// Parse the request body
	var badge database.Badge
	if err := json.NewDecoder(r.Body).Decode(&badge); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the commit ID in the URL matches the one in the request body
	if badge.CommitID != commitID {
		http.Error(w, "Commit ID mismatch", http.StatusBadRequest)
		return
	}

	// Get the user ID from the session
	session, _ := a.authMiddleware.GetSession(r)
	userID, ok := session.Values[auth.SessionUserID].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has access to the badge type
	hasAccess, err := a.db.UserHasAccessToBadgeType(userID, badge.Type)
	if err != nil {
		a.logger.Error("Failed to check badge access", zap.Error(err), zap.String("userID", userID), zap.String("badgeType", badge.Type))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if the user is a superadmin
	user, err := a.db.GetUserByID(userID)
	if err != nil {
		a.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only allow access if the user is a superadmin or has access to the badge type
	if !user.IsSuperadmin && !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Update the badge
	if err := a.db.UpdateBadge(&badge); err != nil {
		a.logger.Error("Failed to update badge", zap.Error(err), zap.String("commitID", commitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the updated badge
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(badge); err != nil {
		a.logger.Error("Failed to encode badge", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// DeleteBadgeHandler handles the request to delete a badge
func (a *API) DeleteBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow DELETE requests
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the commit ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/badges/")
	commitID := strings.Split(path, "/")[0]

	// Get the badge to check its type
	badge, err := a.db.GetBadge(commitID)
	if err != nil {
		a.logger.Error("Failed to get badge", zap.Error(err), zap.String("commitID", commitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if badge == nil {
		http.Error(w, "Badge not found", http.StatusNotFound)
		return
	}

	// Get the user ID from the session
	session, _ := a.authMiddleware.GetSession(r)
	userID, ok := session.Values[auth.SessionUserID].(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has access to the badge type
	hasAccess, err := a.db.UserHasAccessToBadgeType(userID, badge.Type)
	if err != nil {
		a.logger.Error("Failed to check badge access", zap.Error(err), zap.String("userID", userID), zap.String("badgeType", badge.Type))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if the user is a superadmin
	user, err := a.db.GetUserByID(userID)
	if err != nil {
		a.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only allow access if the user is a superadmin or has access to the badge type
	if !user.IsSuperadmin && !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Delete the badge
	if err := a.db.DeleteBadge(commitID); err != nil {
		a.logger.Error("Failed to delete badge", zap.Error(err), zap.String("commitID", commitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}
