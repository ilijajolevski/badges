package auth

import (
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// UIHandler handles the authentication UI
type UIHandler struct {
	logger       *zap.Logger
	authHandler  *Handler
	templatePath string
	templates    map[string]*template.Template
}

// NewUIHandler creates a new authentication UI handler
func NewUIHandler(logger *zap.Logger, authHandler *Handler) *UIHandler {
	h := &UIHandler{
		logger:       logger,
		authHandler:  authHandler,
		templatePath: "templates/auth",
		templates:    make(map[string]*template.Template),
	}

	// Parse templates
	h.templates["login"] = template.Must(template.ParseFiles(filepath.Join(h.templatePath, "login.html")))
	h.templates["dashboard"] = template.Must(template.ParseFiles(filepath.Join(h.templatePath, "dashboard.html")))
	h.templates["auth-error"] = template.Must(template.ParseFiles(filepath.Join(h.templatePath, "auth-error.html")))
	h.templates["callback-success"] = template.Must(template.ParseFiles(filepath.Join(h.templatePath, "callback-success.html")))

	return h
}

// LoginPageHandler handles the login page request
func (h *UIHandler) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	// Render the login page
	data := map[string]interface{}{
		"CurrentYear": time.Now().Year(),
	}

	if err := h.templates["login"].Execute(w, data); err != nil {
		h.logger.Error("Failed to render login template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// DashboardHandler handles the dashboard page request
func (h *UIHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session
	session, err := h.authHandler.sessionStore.Get(r, sessionName)
	if err != nil {
		h.logger.Error("Failed to get session", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user is authenticated
	userID, ok := session.Values[sessionUserID].(string)
	if !ok || userID == "" {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Render the dashboard page
	data := map[string]interface{}{
		"UserID":      userID,
		"Email":       session.Values[sessionEmail],
		"Name":        session.Values[sessionName_],
		"Picture":     session.Values[sessionPicture],
		"CurrentYear": time.Now().Year(),
	}

	// Check if the user is a superadmin
	user, err := h.authHandler.db.GetUserByID(userID)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("userID", userID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user != nil {
		data["IsSuperAdmin"] = user.IsSuperadmin
	}

	// Render the dashboard template
	if err := h.templates["dashboard"].Execute(w, data); err != nil {
		h.logger.Error("Failed to render dashboard template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AuthErrorHandler handles authentication error pages
func (h *UIHandler) AuthErrorHandler(w http.ResponseWriter, r *http.Request, message string, errorDetails string) {
	// Render the error page
	data := map[string]interface{}{
		"Message":      message,
		"ErrorDetails": errorDetails,
		"CurrentYear":  time.Now().Year(),
	}

	// Render the error template
	if err := h.templates["auth-error"].Execute(w, data); err != nil {
		h.logger.Error("Failed to render auth-error template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CallbackSuccessHandler handles the callback success page
func (h *UIHandler) CallbackSuccessHandler(w http.ResponseWriter, r *http.Request) {
	// Render the callback success page
	data := map[string]interface{}{
		"CurrentYear": time.Now().Year(),
	}

	// Render the callback success template
	if err := h.templates["callback-success"].Execute(w, data); err != nil {
		h.logger.Error("Failed to render callback-success template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
