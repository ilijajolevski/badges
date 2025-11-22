package create

import (
    "net/http"
    "time"

    "github.com/finki/badges/internal/cache"
    "github.com/finki/badges/internal/database"
    "go.uber.org/zap"
)

// Handler processes creation of a new certificate/badge by commit ID
type Handler struct {
    db     *database.DB
    logger *zap.Logger
    cache  *cache.Cache
}

func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) *Handler {
    return &Handler{db: db, logger: logger, cache: cache}
}

// ServeHTTP only supports POST; expects form value "commit_id".
// On success creates the badge with minimal defaults and redirects to /edit/{commit_id}.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // read commit_id
    commitID := r.FormValue("commit_id")
    if commitID == "" {
        http.Error(w, "commit_id is required", http.StatusBadRequest)
        return
    }

    // If it already exists, just redirect to edit
    if existing, err := h.db.GetBadge(commitID); err == nil && existing != nil {
        http.Redirect(w, r, "/edit/"+commitID, http.StatusSeeOther)
        return
    }

    // Create with minimal defaults to satisfy NOT NULL columns
    today := time.Now().Format("2006-01-02")
    badge := &database.Badge{
        CommitID:        commitID,
        Type:            "badge",
        Status:          "draft",
        Issuer:          "Unknown",
        IssueDate:       today,
        SoftwareName:    "New Certificate",
        SoftwareVersion: "0.0.0",
    }

    if err := h.db.CreateBadge(badge); err != nil {
        h.logger.Error("failed to create badge", zap.String("commit_id", commitID), zap.Error(err))
        http.Error(w, "Failed to create certificate", http.StatusInternalServerError)
        return
    }

    // Redirect to edit page
    http.Redirect(w, r, "/edit/"+commitID, http.StatusSeeOther)
}
