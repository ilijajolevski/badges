package list

import (
    "encoding/json"
    "fmt"
    "html/template"
    "net/http"
    "strings"
    "time"

    "github.com/finki/badges/internal/auth"
    "github.com/finki/badges/internal/cache"
    "github.com/finki/badges/internal/database"
    "github.com/finki/badges/internal/version"
    "go.uber.org/zap"
)

// TemplateData represents the data passed to the badges list page template
type TemplateData struct {
    Badges      []*BadgeData
    CurrentYear int
    // Permissions
    CanCreate   bool
    Version     string
    Commit      string
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
	    	LabelStyle      template.CSS
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

 // Determine permissions from JWT claims (if any)
    // We consider users with Badges.Write permission as trusted to view drafts
    canSeeDrafts := false
    if claims := auth.GetClaimsFromContext(r.Context()); claims != nil {
        canSeeDrafts = claims.Permissions.Badges.Write
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
      // Hide drafts for unauthenticated/unauthorized users
      if !canSeeDrafts && strings.EqualFold(b.Status, "draft") {
          continue
      }
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
 // Make cache key depend on visibility to avoid leaking drafts
 cacheKey := "badges:list:public"
 if canSeeDrafts {
     cacheKey = "badges:list:priv"
 }
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
        CanCreate:   canSeeDrafts,
        Version:     version.Version,
        Commit:      version.Commit,
    }

	// Convert database badges to template badge data
 for _, badge := range badges {
        // Hide drafts for unauthenticated/unauthorized users
        if !canSeeDrafts && strings.EqualFold(badge.Status, "draft") {
            continue
        }
        certName := ""
        if badge.CertificateName.Valid {
            certName = badge.CertificateName.String
        }

  // Extract colors from custom config with list-specific fallback chain
  colorRight := ""
  borderColor := ""
  textColor := ""
  if cfg, err := badge.GetCustomConfig(); err == nil && cfg != nil {
      // Background
      if cfg.ListColorRight != "" {
          colorRight = cfg.ListColorRight
      } else {
          colorRight = cfg.ColorRight
      }
      // Border
      if cfg.ListBorderColor != "" {
          borderColor = cfg.ListBorderColor
      } else {
          borderColor = cfg.BorderColor
      }
      // Text color
      if cfg.ListTextColor != "" {
          textColor = cfg.ListTextColor
      } else {
          textColor = cfg.TextColorRight
      }
  }

  // Special styling rule: If Certificate Name is "Verified Software Licence",
  // force the silver background, similar to bronze used for "Verified Dependencies".
  if strings.EqualFold(certName, "Verified Software Licence") {
      colorRight = "#E8E8E8"   // silver background
      // Provide a subtle silver border if not already set or to match the style
      borderColor = "#C0C0C0"
  }

  // Build full style string for the label to avoid template concatenation issues
        labelStyle := "display:inline-block;padding:2px 8px;border-radius:4px;"
        if colorRight != "" {
            labelStyle += "background-color: " + colorRight + ";"
        }
        if borderColor != "" {
            labelStyle += " border: 1px solid " + borderColor + ";"
        }
        if textColor != "" {
            labelStyle += " color: " + textColor + ";"
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
            LabelStyle:      template.CSS(labelStyle),
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