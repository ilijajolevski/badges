package list

import (
	"html/template"
	"net/http"
	"time"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// TemplateData represents the data passed to the badges list page template
type TemplateData struct {
	Badges      []*BadgeData
	CurrentYear int
}

// BadgeData represents the data for a single badge in the list
type BadgeData struct {
	CommitID     string
	SoftwareName string
	Status       string
	IssueDate    string
	IsExpired    bool
}

// Handler handles badges list page requests
type Handler struct {
	db       *database.DB
	logger   *zap.Logger
	cache    *cache.Cache
	template *template.Template
}

// NewHandler creates a new badges list handler
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) (*Handler, error) {
	// Parse the template
	tmpl, err := template.ParseFiles("templates/list/list.html")
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

// ServeHTTP handles HTTP requests for the badges list page
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to get from cache first
	cacheKey := "badges:list"
	if cachedData, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(cachedData)
		return
	}

	// Get all badges from database
	badges, err := h.db.ListBadges()
	if err != nil {
		h.logger.Error("Failed to list badges", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := TemplateData{
		Badges:      make([]*BadgeData, 0, len(badges)),
		CurrentYear: time.Now().Year(),
	}

	// Convert database badges to template badge data
	for _, badge := range badges {
		data.Badges = append(data.Badges, &BadgeData{
			CommitID:     badge.CommitID,
			SoftwareName: badge.SoftwareName,
			Status:       badge.Status,
			IssueDate:    badge.IssueDate,
			IsExpired:    badge.IsExpired(),
		})
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}