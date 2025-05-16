package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// API handles user management API endpoints
type API struct {
	db     *database.DB
	logger *zap.Logger
}

// NewAPI creates a new user management API
func NewAPI(db *database.DB, logger *zap.Logger) *API {
	return &API{
		db:     db,
		logger: logger,
	}
}

// ListUsersHandler handles the request to list all users
func (a *API) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Get all users from the database
	users, err := a.db.ListUsers()
	if err != nil {
		a.logger.Error("Failed to list users", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the users as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		a.logger.Error("Failed to encode users", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetUserHandler handles the request to get a user by ID
func (a *API) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the user ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	userID := strings.Split(path, "/")[0]

	// Get the user from the database
	user, err := a.db.GetUserByID(userID)
	if err != nil {
		a.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return the user as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		a.logger.Error("Failed to encode user", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetUserGroupsHandler handles the request to get a user's groups
func (a *API) GetUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the user ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "groups" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID := parts[0]

	// Get the user's groups from the database
	groups, err := a.db.GetUserGroups(userID)
	if err != nil {
		a.logger.Error("Failed to get user groups", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the groups as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groups); err != nil {
		a.logger.Error("Failed to encode groups", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// AddUserToGroupHandler handles the request to add a user to a group
func (a *API) AddUserToGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the user ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "groups" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID := parts[0]

	// Parse the request body
	var request struct {
		GroupID string `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Add the user to the group
	if err := a.db.AddUserToGroup(userID, request.GroupID); err != nil {
		a.logger.Error("Failed to add user to group", zap.Error(err), zap.String("userID", userID), zap.String("groupID", request.GroupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}

// RemoveUserFromGroupHandler handles the request to remove a user from a group
func (a *API) RemoveUserFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow DELETE requests
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the user ID and group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "groups" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	userID := parts[0]
	groupID := parts[2]

	// Remove the user from the group
	if err := a.db.RemoveUserFromGroup(userID, groupID); err != nil {
		a.logger.Error("Failed to remove user from group", zap.Error(err), zap.String("userID", userID), zap.String("groupID", groupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// ListGroupsHandler handles the request to list all domain authority groups
func (a *API) ListGroupsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all groups from the database
	groups, err := a.db.ListGroups()
	if err != nil {
		a.logger.Error("Failed to list groups", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the groups as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groups); err != nil {
		a.logger.Error("Failed to encode groups", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// CreateGroupHandler handles the request to create a new domain authority group
func (a *API) CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var group database.DomainAuthorityGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set the creation time
	group.CreatedAt = time.Now().Format(time.RFC3339)

	// Create the group
	if err := a.db.CreateGroup(&group); err != nil {
		a.logger.Error("Failed to create group", zap.Error(err), zap.String("groupID", group.GroupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the created group
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(group); err != nil {
		a.logger.Error("Failed to encode group", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GetGroupHandler handles the request to get a group by ID
func (a *API) GetGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	groupID := strings.Split(path, "/")[0]

	// Get the group from the database
	group, err := a.db.GetGroupByID(groupID)
	if err != nil {
		a.logger.Error("Failed to get group", zap.Error(err), zap.String("groupID", groupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if group == nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	// Return the group as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(group); err != nil {
		a.logger.Error("Failed to encode group", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// UpdateGroupHandler handles the request to update a group
func (a *API) UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow PUT requests
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	groupID := strings.Split(path, "/")[0]

	// Parse the request body
	var group database.DomainAuthorityGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the group ID in the URL matches the one in the request body
	if group.GroupID != groupID {
		http.Error(w, "Group ID mismatch", http.StatusBadRequest)
		return
	}

	// Update the group
	if err := a.db.UpdateGroup(&group); err != nil {
		a.logger.Error("Failed to update group", zap.Error(err), zap.String("groupID", groupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return the updated group
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(group); err != nil {
		a.logger.Error("Failed to encode group", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// DeleteGroupHandler handles the request to delete a group
func (a *API) DeleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow DELETE requests
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	groupID := strings.Split(path, "/")[0]

	// Delete the group
	if err := a.db.DeleteGroup(groupID); err != nil {
		a.logger.Error("Failed to delete group", zap.Error(err), zap.String("groupID", groupID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// ListGroupBadgesHandler handles the request to list all badges assigned to a group
func (a *API) ListGroupBadgesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "badges" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	groupID := parts[0]

	// Get the badges assigned to the group
	badges, err := a.db.GetGroupBadges(groupID)
	if err != nil {
		a.logger.Error("Failed to get group badges", zap.Error(err), zap.String("groupID", groupID))
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

// AssignBadgeToGroupHandler handles the request to assign a badge to a group
func (a *API) AssignBadgeToGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the group ID from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "badges" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	groupID := parts[0]

	// Parse the request body
	var request struct {
		BadgeType string `json:"badge_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		a.logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Assign the badge to the group
	if err := a.db.AssignBadgeToGroup(request.BadgeType, groupID); err != nil {
		a.logger.Error("Failed to assign badge to group", zap.Error(err), zap.String("groupID", groupID), zap.String("badgeType", request.BadgeType))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}

// RemoveBadgeFromGroupHandler handles the request to remove a badge from a group
func (a *API) RemoveBadgeFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow DELETE requests
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the group ID and badge type from the URL
	path := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "badges" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	groupID := parts[0]
	badgeType := parts[2]

	// Remove the badge from the group
	if err := a.db.RemoveBadgeFromGroup(badgeType, groupID); err != nil {
		a.logger.Error("Failed to remove badge from group", zap.Error(err), zap.String("groupID", groupID), zap.String("badgeType", badgeType))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}
