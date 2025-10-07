package home

import (
	"html/template"
	"net/http"
	"time"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// TemplateData represents the data passed to the home page template
type TemplateData struct {
	CurrentYear int
}

// Handler handles home page requests
type Handler struct {
	db       *database.DB
	logger   *zap.Logger
	cache    *cache.Cache
	template *template.Template
}

// NewHandler creates a new home handler
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) (*Handler, error) {
	// Parse the template
	tmpl, err := template.ParseFiles("templates/home/index.html")
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

// ServeHTTP handles HTTP requests for the home page
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only serve exact root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Try to get from cache first
	cacheKey := "home:index"
	if cachedData, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(cachedData)
		return
	}

	// Prepare template data
	data := TemplateData{
		CurrentYear: time.Now().Year(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render home template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
