package list

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
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
	CommitID        string
	SoftwareName    string
	CertificateName string
	Status          string
	IssueDate       string
	IsExpired       bool
	ColorRight      string
	BorderColor     string
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
	accept := r.Header.Get("Accept")
	wantsJSON := strings.Contains(accept, "application/json")

	// Query parameter override (?format=json or ?format=html)
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "json" {
		wantsJSON = true
	} else if format == "html" {
		wantsJSON = false
	}

	if wantsJSON {
		// Return JSON representation of certificates
		badges, err := h.db.ListBadges()
		if err != nil {
			h.logger.Error("Failed to list badges", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Define minimal JSON view model
		type CertificateJSON struct {
			CertID          string `json:"cert_id"`
			SoftwareName    string `json:"software_name"`
			CertificateName string `json:"certificate_name,omitempty"`
			Status          string `json:"status"`
			IssueDate       string `json:"issue_date"`
			IsExpired       bool   `json:"is_expired"`
			DetailsLink     string `json:"details_link"`
		}

		result := make([]CertificateJSON, 0, len(badges))
		for _, b := range badges {
			certName := ""
			if b.CertificateName.Valid {
				certName = b.CertificateName.String
			}
			result = append(result, CertificateJSON{
				CertID:          b.CommitID,
				SoftwareName:    b.SoftwareName,
				CertificateName: certName,
				Status:          b.Status,
				IssueDate:       b.IssueDate,
				IsExpired:       b.IsExpired(),
				DetailsLink:     fmt.Sprintf("https://certificates.software.geant.org/details/%s", b.CommitID),
			})
		}

		payload, err := json.Marshal(result)
		if err != nil {
			h.logger.Error("Failed to marshal certificates JSON", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
		return
	}

	// Default: Return existing HTML page
	// Try to get from cache first (existing behavior)
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
		certName := ""
		if badge.CertificateName.Valid {
			certName = badge.CertificateName.String
		}

		// Extract colors from custom config with safe defaults
		colorRight := ""
		borderColor := ""
		if cfg, err := badge.GetCustomConfig(); err == nil && cfg != nil {
			colorRight = cfg.ColorRight
			borderColor = cfg.BorderColor
		}

		data.Badges = append(data.Badges, &BadgeData{
			CommitID:        badge.CommitID,
			SoftwareName:    badge.SoftwareName,
			CertificateName: certName,
			Status:          badge.Status,
			IssueDate:       badge.IssueDate,
			IsExpired:       badge.IsExpired(),
			ColorRight:      colorRight,
			BorderColor:     borderColor,
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