package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// Handler handles authentication requests
type Handler struct {
	DB     *database.DB
	Logger *zap.Logger
}

// NewHandler creates a new authentication handler
func NewHandler(db *database.DB, logger *zap.Logger) *Handler {
	return &Handler{
		DB:     db,
		Logger: logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      struct {
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
	} `json:"user"`
}

// Login handles user authentication and returns a JWT token
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Only handle POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Get user from database
	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil {
		h.Logger.Error("Failed to get user", zap.Error(err))
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Check if user exists
	if user == nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Check if user is active
	if user.Status != "active" {
		http.Error(w, "Account is not active", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// Increment failed attempts
		if err := h.DB.UpdateUserFailedAttempts(user.UserID, user.FailedAttempts+1); err != nil {
			h.Logger.Error("Failed to update failed attempts", zap.Error(err))
		}

		// Check if account should be locked
		if user.FailedAttempts+1 >= 5 {
			// Lock account
			user.Status = "locked"
			if err := h.DB.UpdateUser(user); err != nil {
				h.Logger.Error("Failed to lock account", zap.Error(err))
			}
			http.Error(w, "Account has been locked due to too many failed attempts", http.StatusUnauthorized)
			return
		}

		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Reset failed attempts
	if user.FailedAttempts > 0 {
		if err := h.DB.UpdateUserFailedAttempts(user.UserID, 0); err != nil {
			h.Logger.Error("Failed to reset failed attempts", zap.Error(err))
		}
	}

	// Update last login
	if err := h.DB.UpdateUserLastLogin(user.UserID, time.Now()); err != nil {
		h.Logger.Error("Failed to update last login", zap.Error(err))
	}

	// Get role
	role, err := h.DB.GetRole(user.RoleID)
	if err != nil {
		h.Logger.Error("Failed to get role", zap.Error(err))
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Get permissions
	permissions, err := role.GetPermissions()
	if err != nil {
		h.Logger.Error("Failed to get permissions", zap.Error(err))
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Convert permissions to map
	permissionsMap := map[string]interface{}{
		"badges": map[string]interface{}{
			"read":   permissions.Badges.Read,
			"write":  permissions.Badges.Write,
			"delete": permissions.Badges.Delete,
		},
		"users": map[string]interface{}{
			"read":   permissions.Users.Read,
			"write":  permissions.Users.Write,
			"delete": permissions.Users.Delete,
		},
		"api_keys": map[string]interface{}{
			"read":   permissions.APIKeys.Read,
			"write":  permissions.APIKeys.Write,
			"delete": permissions.APIKeys.Delete,
		},
	}

	// Generate JWT token
	token, expiresAt, err := GenerateToken(user.UserID, user.Username, user.Email, role.Name, permissionsMap)
	if err != nil {
		h.Logger.Error("Failed to generate token", zap.Error(err))
		http.Error(w, "Failed to authenticate", http.StatusInternalServerError)
		return
	}

	// Create response
	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	resp.User.UserID = user.UserID
	resp.User.Username = user.Username
	resp.User.Email = user.Email
	resp.User.FirstName = user.FirstName
	resp.User.LastName = user.LastName
	resp.User.Role = role.Name

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}