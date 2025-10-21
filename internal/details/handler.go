package details

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// certificateGuideLinks maps certificate names to their corresponding detailed guide URLs.
var certificateGuideLinks = map[string]string{
	"Self-Assessed Dependencies": "https://wiki.geant.org/spaces/GSD/pages/1190199425/Detailed+Guide+Self-Assessed+Dependencies+Certificate",
	"Verified Dependencies":      "https://wiki.geant.org/spaces/GSD/pages/1190199427/Quick+Guide+Verified+Dependencies+Certificate",
	"Verified Software Licence":  "https://wiki.geant.org/spaces/GSD/pages/1190199433/Quick+Guide+Verified+Software+Licence+Certificate",
	"Software Licence Assurance": "https://wiki.geant.org/spaces/GSD/pages/1190199436/Quick+Guide+Software+Licence+Assurance+Certificate",
}

// TemplateData represents the data passed to the details page template
type TemplateData struct {
	CommitID            string
	Type                string
	Status              string
	Issuer              string
	IssueDate           string
	SoftwareName        string
	SoftwareVersion     string
	SoftwareURL         string
	Notes               string
	ExpiryDate          string
	IssuerURL           string
	LastReview          string
	IsExpired           bool
	CurrentYear         int
	CoveredVersion      string
	RepositoryLink      string
	PublicNote          string
	InternalNote        string
	ContactDetails      string
	CertificateName     string
	CertificateGuideURL string
	SpecialtyDomain     string
	SoftwareSCID        string
	SoftwareSCURL       string
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Content negotiation: JSON vs HTML
	accept := r.Header.Get("Accept")
	wantsJSON := strings.Contains(accept, "application/json")

	if wantsJSON {
		// Get badge from database
		badge, err := h.db.GetBadge(commitID)
		if err != nil {
			h.logger.Error("Failed to get badge", zap.Error(err), zap.String("commit_id", commitID))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if badge == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Build comprehensive JSON response
		type CertificateDetailsJSON struct {
			CertID              string `json:"cert_id"`
			Type                string `json:"type"`
			Status              string `json:"status"`
			Issuer              string `json:"issuer"`
			IssueDate           string `json:"issue_date"`
			SoftwareName        string `json:"software_name"`
			SoftwareVersion     string `json:"software_version"`
			SoftwareURL         string `json:"software_url,omitempty"`
			Notes               string `json:"notes,omitempty"`
			ExpiryDate          string `json:"expiry_date,omitempty"`
			IssuerURL           string `json:"issuer_url,omitempty"`
			LastReview          string `json:"last_review,omitempty"`
			IsExpired           bool   `json:"is_expired"`
			CoveredVersion      string `json:"covered_version,omitempty"`
			RepositoryLink      string `json:"repository_link,omitempty"`
			PublicNote          string `json:"public_note,omitempty"`
			ContactDetails      string `json:"contact_details,omitempty"`
			CertificateName     string `json:"certificate_name,omitempty"`
			CertificateGuideURL string `json:"certificate_guide_url,omitempty"`
			SpecialtyDomain     string `json:"specialty_domain,omitempty"`
			SoftwareSCID        string `json:"software_sc_id,omitempty"`
			SoftwareSCURL       string `json:"software_sc_url,omitempty"`
		}

		resp := CertificateDetailsJSON{
			CertID:          badge.CommitID,
			Type:            badge.Type,
			Status:          badge.Status,
			Issuer:          badge.Issuer,
			IssueDate:       badge.IssueDate,
			SoftwareName:    badge.SoftwareName,
			SoftwareVersion: badge.SoftwareVersion,
			IsExpired:       badge.IsExpired(),
		}

		if badge.SoftwareURL.Valid {
			resp.SoftwareURL = badge.SoftwareURL.String
		}
		if badge.Notes.Valid {
			resp.Notes = badge.Notes.String
		}
		if badge.ExpiryDate.Valid {
			resp.ExpiryDate = badge.ExpiryDate.String
		}
		if badge.IssuerURL.Valid {
			resp.IssuerURL = badge.IssuerURL.String
		}
		if badge.LastReview.Valid {
			resp.LastReview = badge.LastReview.String
		}
		if badge.CoveredVersion.Valid {
			resp.CoveredVersion = badge.CoveredVersion.String
		}
		if badge.RepositoryLink.Valid {
			resp.RepositoryLink = badge.RepositoryLink.String
		}
		if badge.PublicNote.Valid {
			resp.PublicNote = badge.PublicNote.String
		}
		if badge.ContactDetails.Valid {
			resp.ContactDetails = badge.ContactDetails.String
		}
		if badge.CertificateName.Valid {
			resp.CertificateName = badge.CertificateName.String
			if url, ok := certificateGuideLinks[resp.CertificateName]; ok {
				resp.CertificateGuideURL = url
			}
		}
		if badge.SpecialtyDomain.Valid {
			resp.SpecialtyDomain = badge.SpecialtyDomain.String
		}
		if badge.SoftwareSCID.Valid {
			resp.SoftwareSCID = badge.SoftwareSCID.String
		}
		if badge.SoftwareSCURL.Valid {
			resp.SoftwareSCURL = badge.SoftwareSCURL.String
		}

		payload, err := json.Marshal(resp)
		if err != nil {
			h.logger.Error("Failed to marshal details JSON", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
		return
	}

	// HTML path (existing behavior)
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if badge == nil {
		w.WriteHeader(http.StatusNotFound)
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
		if url, ok := certificateGuideLinks[data.CertificateName]; ok {
			data.CertificateGuideURL = url
		}
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
