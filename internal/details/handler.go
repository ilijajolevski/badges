package details

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// TemplateData represents the data passed to the details page template
type TemplateData struct {
	CommitID        string
	Type            string
	Status          string
	Issuer          string
	IssueDate       string
	SoftwareName    string
	SoftwareVersion string
	SoftwareURL     string
	Notes           string
	ExpiryDate      string
	IssuerURL       string
	LastReview      string
	IsExpired       bool
	CurrentYear     int
	CoveredVersion  string
	RepositoryLink  string
	PublicNote      string
	InternalNote    string
	ContactDetails  string
	CertificateName string
	SpecialtyDomain string
	SoftwareSCID    string
	SoftwareSCURL   string
}

// Handler handles details page requests
type Handler struct {
	db       *database.DB
	logger   *zap.Logger
	cache    *cache.Cache
	template *template.Template
}

// NewHandler creates a new details handler
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) (*Handler, error) {
	// Parse the template
	tmpl, err := template.ParseFiles("templates/details/details.html")
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

// ServeHTTP handles HTTP requests for the details page
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract commit ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/details/")
	commitID := strings.Split(path, "/")[0]

	if commitID == "" {
		http.Error(w, "Missing commit ID", http.StatusBadRequest)
		return
	}

	// Try to get from cache first
	cacheKey := "details:" + commitID
	if cachedData, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(cachedData)
		return
	}

	// Get badge from database
	badge, err := h.db.GetBadge(commitID)
	if err != nil {
		h.logger.Error("Failed to get badge", zap.Error(err), zap.String("commit_id", commitID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if badge == nil {
		http.Error(w, "Badge not found", http.StatusNotFound)
		return
	}

	// Prepare template data
	data := TemplateData{
		CommitID:        badge.CommitID,
		Type:            badge.Type,
		Status:          badge.Status,
		Issuer:          badge.Issuer,
		IssueDate:       badge.IssueDate,
		SoftwareName:    badge.SoftwareName,
		SoftwareVersion: badge.SoftwareVersion,
		CurrentYear:     time.Now().Year(),
		IsExpired:       badge.IsExpired(),
	}

	// Add optional fields if they exist
	if badge.Notes.Valid {
		data.Notes = badge.Notes.String
	}

	if badge.ExpiryDate.Valid {
		data.ExpiryDate = badge.ExpiryDate.String
	}

	if badge.IssuerURL.Valid {
		data.IssuerURL = badge.IssuerURL.String
	}

	if badge.SoftwareURL.Valid {
		data.SoftwareURL = badge.SoftwareURL.String
	}

	if badge.LastReview.Valid {
		data.LastReview = badge.LastReview.String
	}

	if badge.CoveredVersion.Valid {
		data.CoveredVersion = badge.CoveredVersion.String
	}

	if badge.RepositoryLink.Valid {
		data.RepositoryLink = badge.RepositoryLink.String
	}

	if badge.PublicNote.Valid {
		data.PublicNote = badge.PublicNote.String
	}

	if badge.InternalNote.Valid {
		data.InternalNote = badge.InternalNote.String
	}

	if badge.ContactDetails.Valid {
		data.ContactDetails = badge.ContactDetails.String
	}

	if badge.CertificateName.Valid {
		data.CertificateName = badge.CertificateName.String
	}

	if badge.SpecialtyDomain.Valid {
		data.SpecialtyDomain = badge.SpecialtyDomain.String
	}

	if badge.SoftwareSCID.Valid {
		data.SoftwareSCID = badge.SoftwareSCID.String
	}

	if badge.SoftwareSCURL.Valid {
		data.SoftwareSCURL = badge.SoftwareSCURL.String
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
