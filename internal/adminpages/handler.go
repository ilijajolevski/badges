// Package adminpages serves the authenticated-only admin screens (backup, restore,
// change password). Each screen is a standalone HTML page that drives the existing
// JSON APIs (/api/backup, /api/restore, /api/auth/password). Pages are gated on an
// authenticated session and redirect unauthenticated visitors to /admin to log in.
package adminpages

import (
	"html/template"
	"net/http"
	"time"

	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/version"
	"go.uber.org/zap"
)

// TemplateData is the data passed to every admin page template.
type TemplateData struct {
	CurrentYear int
	Version     string
	Commit      string
}

// Handler renders a single auth-gated admin page.
type Handler struct {
	logger   *zap.Logger
	path     string
	template *template.Template
}

// NewHandler parses templatePath and returns a handler that only serves requests
// for exactly path (e.g. "/backup") to authenticated users.
func NewHandler(logger *zap.Logger, path, templatePath string) (*Handler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}
	return &Handler{logger: logger, path: path, template: tmpl}, nil
}

// ServeHTTP renders the page, redirecting unauthenticated visitors to /admin.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.path {
		http.NotFound(w, r)
		return
	}

	if auth.GetClaimsFromContext(r.Context()) == nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	data := TemplateData{
		CurrentYear: time.Now().Year(),
		Version:     version.Version,
		Commit:      version.Commit,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render admin page", zap.String("path", h.path), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
