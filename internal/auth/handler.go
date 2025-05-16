package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/finki/badges/internal/config"
	"github.com/finki/badges/internal/database"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

const (
	sessionName     = "badge-service-session"
	sessionUserID   = "user_id"
	SessionUserID   = sessionUserID // Exported version for use in other packages
	sessionEmail    = "email"
	sessionName_    = "name"
	sessionPicture  = "picture"
	sessionState    = "state"
)

// Handler handles authentication requests
type Handler struct {
	provider      *Provider
	db            *database.DB
	sessionStore  *sessions.CookieStore
	sessionMaxAge int
	logger        *zap.Logger
}

// NewHandler creates a new authentication handler
func NewHandler(ctx context.Context, cfg *config.Config, db *database.DB, logger *zap.Logger) (*Handler, error) {
	// Create OIDC provider
	provider, err := NewProvider(ctx, &cfg.Auth, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Create session store
	sessionStore := sessions.NewCookieStore([]byte(cfg.Auth.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(cfg.Auth.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   cfg.LogLevel == "production", // Secure in production only
		SameSite: http.SameSiteLaxMode,
	}

	return &Handler{
		provider:      provider,
		db:            db,
		sessionStore:  sessionStore,
		sessionMaxAge: int(cfg.Auth.SessionDuration.Seconds()),
		logger:        logger,
	}, nil
}

// LoginHandler handles the login request
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a random state
	state, err := generateRandomState()
	if err != nil {
		h.logger.Error("Failed to generate random state", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store the state in the session
	session, _ := h.sessionStore.Get(r, sessionName)
	session.Values[sessionState] = state
	if err := session.Save(r, w); err != nil {
		h.logger.Error("Failed to save session", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect to the auth URL
	authURL := h.provider.GetAuthURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler handles the callback from the OIDC provider
func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the state from the session
	session, err := h.sessionStore.Get(r, sessionName)
	if err != nil {
		h.logger.Error("Failed to get session", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify the state
	state := r.URL.Query().Get("state")
	sessionState, ok := session.Values[sessionState].(string)
	if !ok || state != sessionState {
		h.logger.Error("Invalid state", zap.String("state", state), zap.String("sessionState", sessionState))
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	// Exchange the code for a token
	code := r.URL.Query().Get("code")
	token, err := h.provider.Exchange(r.Context(), code)
	if err != nil {
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	// Get user info
	userInfo, err := h.provider.GetUserInfo(r.Context(), token)
	if err != nil {
		h.logger.Error("Failed to get user info", zap.Error(err))
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Check if the user exists in the database
	user, err := h.db.GetUserByEmail(userInfo.Email)
	if err != nil {
		h.logger.Error("Failed to get user by email", zap.Error(err), zap.String("email", userInfo.Email))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If the user doesn't exist, create a new user
	if user == nil {
		user = &database.User{
			UserID:    userInfo.UserID,
			Email:     userInfo.Email,
			Name:      userInfo.Name,
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		// Check if this is the initial superadmin
		if userInfo.Email == h.provider.config.InitialSuperAdminEmail {
			user.IsSuperadmin = true
		}

		// Create the user
		if err := h.db.CreateUser(user); err != nil {
			h.logger.Error("Failed to create user", zap.Error(err), zap.String("email", userInfo.Email))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Add the user to the demo group
		if err := h.db.AddUserToGroup(user.UserID, "demo_group"); err != nil {
			h.logger.Error("Failed to add user to demo group", zap.Error(err), zap.String("email", userInfo.Email))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Update the last login time
	if err := h.db.UpdateUserLastLogin(user.UserID); err != nil {
		h.logger.Error("Failed to update user last login", zap.Error(err), zap.String("email", userInfo.Email))
		// Continue anyway, this is not critical
	}

	// Store user info in the session
	session.Values[sessionUserID] = user.UserID
	session.Values[sessionEmail] = user.Email
	session.Values[sessionName_] = user.Name
	session.Values[sessionPicture] = userInfo.Picture
	session.Options.MaxAge = h.sessionMaxAge
	if err := session.Save(r, w); err != nil {
		h.logger.Error("Failed to save session", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect to the dashboard
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// LogoutHandler handles the logout request
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session
	session, _ := h.sessionStore.Get(r, sessionName)
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		h.logger.Error("Failed to save session", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect to the home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// generateRandomState generates a random state string
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
