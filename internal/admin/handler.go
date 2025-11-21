package admin

import (
    "html/template"
    "net/http"
    "time"

    "github.com/finki/badges/internal/cache"
    "github.com/finki/badges/internal/database"
    "go.uber.org/zap"
)

// TemplateData represents the data passed to the admin page template
type TemplateData struct {
    CurrentYear int
}

// Handler serves the /admin page
type Handler struct {
    db       *database.DB
    logger   *zap.Logger
    cache    *cache.Cache
    template *template.Template
}

// NewHandler creates a new admin handler
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) (*Handler, error) {
    tmpl, err := template.ParseFiles("templates/admin/index.html")
    if err != nil {
        return nil, err
    }

    return &Handler{
        db:       db,
        logger:   logger,
        cache:    cache,
        template: tmpl,
    }, nil
}

// ServeHTTP renders the admin page
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/admin" {
        http.NotFound(w, r)
        return
    }

    data := TemplateData{CurrentYear: time.Now().Year()}

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := h.template.Execute(w, data); err != nil {
        h.logger.Error("Failed to render admin template", zap.Error(err))
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
}
