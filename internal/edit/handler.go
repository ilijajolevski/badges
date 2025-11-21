package edit

import (
    "database/sql"
    "html/template"
    "net/http"
    "strings"
    "time"

    "github.com/finki/badges/internal/auth"
    "github.com/finki/badges/internal/cache"
    "github.com/finki/badges/internal/database"
    "go.uber.org/zap"
)

// TemplateData holds the data shown on the edit page
type TemplateData struct {
    CurrentYear int
    Badge       *database.Badge
    // Permissions
    CanDelete bool
}

// Handler serves the /edit/{id} page and processes updates/deletes
type Handler struct {
    db       *database.DB
    logger   *zap.Logger
    cache    *cache.Cache
    template *template.Template
}

func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) (*Handler, error) {
    tmpl, err := template.ParseFiles("templates/edit/edit.html")
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

// ServeHTTP renders edit form on GET and processes updates/deletes on POST
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Must be under /edit/
    if !strings.HasPrefix(r.URL.Path, "/edit/") {
        http.NotFound(w, r)
        return
    }

    // Only authenticated users with permissions can do anything. We use OptionalJWTFromCookie upstream
    // and enforce permissions quietly here (render empty page if missing/unauthenticated).
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        // Render empty page as per requirements
        w.WriteHeader(http.StatusOK)
        return
    }

    // Extract ID
    commitID := strings.TrimPrefix(r.URL.Path, "/edit/")
    if commitID == "" {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // Load badge
    badge, err := h.db.GetBadge(commitID)
    if err != nil || badge == nil {
        if err != nil {
            h.logger.Error("failed to load badge for edit", zap.String("commit_id", commitID), zap.Error(err))
        }
        // Empty page on missing or error (avoid leaking existence details)
        w.WriteHeader(http.StatusOK)
        return
    }

    // Permission flags
    canWrite := claims.Permissions.Badges.Write
    canDelete := claims.Permissions.Badges.Delete

    switch r.Method {
    case http.MethodGet:
        if !canWrite {
            w.WriteHeader(http.StatusOK)
            return
        }
        data := TemplateData{CurrentYear: time.Now().Year(), Badge: badge, CanDelete: canDelete}
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        if err := h.template.Execute(w, data); err != nil {
            h.logger.Error("failed to render edit template", zap.Error(err))
            http.Error(w, "Internal server error", http.StatusInternalServerError)
            return
        }

    case http.MethodPost:
        // Determine action
        action := r.FormValue("action")
        if action == "delete" {
            if !canDelete {
                w.WriteHeader(http.StatusOK)
                return
            }
            if err := h.db.DeleteBadge(commitID); err != nil {
                h.logger.Error("failed to delete badge", zap.String("commit_id", commitID), zap.Error(err))
                http.Error(w, "Failed to delete", http.StatusInternalServerError)
                return
            }
            http.Redirect(w, r, "/", http.StatusSeeOther)
            return
        }

        if !canWrite {
            w.WriteHeader(http.StatusOK)
            return
        }

        // Parse form values and update badge
        // Simple helpers for nullable strings
        toNull := func(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

        // Required/basic fields
        badge.Status = r.FormValue("status")
        badge.Issuer = r.FormValue("issuer")
        badge.IssueDate = r.FormValue("issue_date")
        badge.SoftwareName = r.FormValue("software_name")
        badge.SoftwareVersion = r.FormValue("software_version")
        badge.SoftwareURL = toNull(r.FormValue("software_url"))
        badge.Notes = toNull(r.FormValue("notes"))
        badge.ExpiryDate = toNull(r.FormValue("expiry_date"))
        badge.IssuerURL = toNull(r.FormValue("issuer_url"))
        badge.LastReview = toNull(r.FormValue("last_review"))
        badge.CoveredVersion = toNull(r.FormValue("covered_version"))
        badge.RepositoryLink = toNull(r.FormValue("repository_link"))
        badge.PublicNote = toNull(r.FormValue("public_note"))
        badge.InternalNote = toNull(r.FormValue("internal_note"))
        badge.ContactDetails = toNull(r.FormValue("contact_details"))
        badge.CertificateName = toNull(r.FormValue("certificate_name"))
        badge.SpecialtyDomain = toNull(r.FormValue("specialty_domain"))
        badge.SoftwareSCID = toNull(r.FormValue("software_sc_id"))
        badge.SoftwareSCURL = toNull(r.FormValue("software_sc_url"))

        if err := h.db.UpdateBadge(badge); err != nil {
            h.logger.Error("failed to update badge", zap.String("commit_id", commitID), zap.Error(err))
            http.Error(w, "Failed to update", http.StatusInternalServerError)
            return
        }

        // Redirect to details page after update
        http.Redirect(w, r, "/details/"+commitID, http.StatusSeeOther)
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}
