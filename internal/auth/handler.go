package auth

import (
    "encoding/json"
    "net/http"
    "strings"
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

// writeJSONError writes a consistent JSON error response with a given HTTP status
func writeJSONError(w http.ResponseWriter, status int, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(map[string]string{
        "error": message,
    })
}

// Login handles user authentication and returns a JWT token
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    // Only handle POST requests
    if r.Method != http.MethodPost {
        writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

	// Parse request body
	var req LoginRequest
 if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

	// Validate request
 if req.Username == "" || req.Password == "" {
        writeJSONError(w, http.StatusBadRequest, "Username and password are required")
        return
    }

 // Get user from database (allow username or email)
 var (
     user *database.User
     err  error
 )
 if strings.Contains(req.Username, "@") {
     user, err = h.DB.GetUserByEmail(req.Username)
 } else {
     user, err = h.DB.GetUserByUsername(req.Username)
 }
 if err != nil {
        h.Logger.Error("Failed to get user", zap.Error(err))
        writeJSONError(w, http.StatusUnauthorized, "Invalid username or password")
        return
    }

	// Check if user exists
 if user == nil {
        writeJSONError(w, http.StatusUnauthorized, "Invalid username or password")
        return
    }

	// Check if user is active
 if user.Status != "active" {
        writeJSONError(w, http.StatusUnauthorized, "Account is not active")
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
            writeJSONError(w, http.StatusUnauthorized, "Account has been locked due to too many failed attempts")
            return
        }

        writeJSONError(w, http.StatusUnauthorized, "Invalid username or password")
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
        writeJSONError(w, http.StatusInternalServerError, "Failed to authenticate")
        return
    }

	// Get permissions
	permissions, err := role.GetPermissions()
 if err != nil {
        h.Logger.Error("Failed to get permissions", zap.Error(err))
        writeJSONError(w, http.StatusInternalServerError, "Failed to authenticate")
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
        writeJSONError(w, http.StatusInternalServerError, "Failed to authenticate")
        return
    }

 // Set HttpOnly cookie with the JWT for browser sessions
 http.SetCookie(w, &http.Cookie{
     Name:     "jwt",
     Value:    token,
     Path:     "/",
     Expires:  expiresAt,
     HttpOnly: true,
     SameSite: http.SameSiteLaxMode,
     // Secure: true, // enable when using HTTPS
 })

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

// Logout clears the JWT cookie for browser sessions
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Invalidate the cookie by setting it to expire in the past
    http.SetCookie(w, &http.Cookie{
        Name:     "jwt",
        Value:    "",
        Path:     "/",
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
        // Secure: true, // enable when using HTTPS
    })

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

// Session returns the current authenticated session info based on JWT cookie
func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
    // Read cookie
    c, err := r.Cookie("jwt")
    if err != nil || c.Value == "" {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "authenticated": false,
        })
        return
    }

    claims, err := ValidateToken(c.Value)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "authenticated": false,
        })
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "authenticated": true,
        "user": map[string]string{
            "user_id":  claims.UserID,
            "username": claims.Username,
            "email":    claims.Email,
            "role":     claims.Role,
        },
        "expires_at": claims.ExpiresAt.Time,
    })
}