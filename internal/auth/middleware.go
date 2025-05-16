package auth

import (
	"net/http"

	"github.com/finki/badges/internal/database"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

// Middleware handles authentication and authorization
type Middleware struct {
	sessionStore *sessions.CookieStore
	db           *database.DB
	logger       *zap.Logger
}

// GetSession retrieves the session from the request
func (m *Middleware) GetSession(r *http.Request) (*sessions.Session, error) {
	return m.sessionStore.Get(r, sessionName)
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(sessionStore *sessions.CookieStore, db *database.DB, logger *zap.Logger) *Middleware {
	return &Middleware{
		sessionStore: sessionStore,
		db:           db,
		logger:       logger,
	}
}

// RequireAuthentication middleware ensures the user is authenticated
func (m *Middleware) RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session
		session, err := m.sessionStore.Get(r, sessionName)
		if err != nil {
			m.logger.Error("Failed to get session", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the user is authenticated
		userID, ok := session.Values[sessionUserID].(string)
		if !ok || userID == "" {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// RequireSuperAdmin middleware ensures the user is a superadmin
func (m *Middleware) RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session
		session, err := m.sessionStore.Get(r, sessionName)
		if err != nil {
			m.logger.Error("Failed to get session", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the user is authenticated
		userID, ok := session.Values[sessionUserID].(string)
		if !ok || userID == "" {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		// Get the user from the database
		user, err := m.db.GetUserByID(userID)
		if err != nil {
			m.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check if the user is a superadmin
		if user == nil || !user.IsSuperadmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// RequireGroupMembership middleware ensures the user is a member of the specified group
func (m *Middleware) RequireGroupMembership(groupID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the session
			session, err := m.sessionStore.Get(r, sessionName)
			if err != nil {
				m.logger.Error("Failed to get session", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if the user is authenticated
			userID, ok := session.Values[sessionUserID].(string)
			if !ok || userID == "" {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Get the user from the database
			user, err := m.db.GetUserByID(userID)
			if err != nil {
				m.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Superadmins can access any group
			if user != nil && user.IsSuperadmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check if the user is a member of the group
			isMember, err := m.db.IsUserInGroup(userID, groupID)
			if err != nil {
				m.logger.Error("Failed to check group membership", zap.Error(err), zap.String("userID", userID), zap.String("groupID", groupID))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !isMember {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// RequireBadgeGroupAccess middleware ensures the user has access to the badge type
func (m *Middleware) RequireBadgeGroupAccess(badgeType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the session
			session, err := m.sessionStore.Get(r, sessionName)
			if err != nil {
				m.logger.Error("Failed to get session", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if the user is authenticated
			userID, ok := session.Values[sessionUserID].(string)
			if !ok || userID == "" {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Get the user from the database
			user, err := m.db.GetUserByID(userID)
			if err != nil {
				m.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Superadmins can access any badge
			if user != nil && user.IsSuperadmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check if the user has access to the badge type
			hasAccess, err := m.db.UserHasAccessToBadgeType(userID, badgeType)
			if err != nil {
				m.logger.Error("Failed to check badge access", zap.Error(err), zap.String("userID", userID), zap.String("badgeType", badgeType))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !hasAccess {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
