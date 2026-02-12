package backup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// Handler handles backup and restore HTTP requests.
type Handler struct {
	db     *database.DB
	logger *zap.Logger
	cache  *cache.Cache
}

// NewHandler creates a new backup Handler.
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) *Handler {
	return &Handler{db: db, logger: logger, cache: cache}
}

// Backup exports the entire database as a JSON download.
func (h *Handler) Backup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Defense-in-depth: verify admin role even though middleware already checks permission
	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	roles, err := h.db.ListRoles()
	if err != nil {
		h.logger.Error("backup: failed to list roles", zap.Error(err))
		http.Error(w, "Failed to read roles", http.StatusInternalServerError)
		return
	}

	users, err := h.db.ListUsers()
	if err != nil {
		h.logger.Error("backup: failed to list users", zap.Error(err))
		http.Error(w, "Failed to read users", http.StatusInternalServerError)
		return
	}

	apiKeys, err := h.db.ListAPIKeys()
	if err != nil {
		h.logger.Error("backup: failed to list API keys", zap.Error(err))
		http.Error(w, "Failed to read API keys", http.StatusInternalServerError)
		return
	}

	badges, err := h.db.ListBadges()
	if err != nil {
		h.logger.Error("backup: failed to list badges", zap.Error(err))
		http.Error(w, "Failed to read badges", http.StatusInternalServerError)
		return
	}

	doc := BackupDocument{
		Metadata: BackupMetadata{
			Version:     1,
			CreatedAt:   time.Now().UTC().Format(timeFormat),
			CreatedBy:   claims.Username,
			Application: "CertifyHub",
		},
		Data: BackupData{
			Roles:   rolesToDTOs(roles),
			Users:   usersToDTOs(users),
			APIKeys: apiKeysToDTOs(apiKeys),
			Badges:  badgesToDTOs(badges),
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		h.logger.Error("backup: failed to marshal JSON", zap.Error(err))
		http.Error(w, "Failed to generate backup", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("certifyhub-backup-%s.json", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(data)
}

// Restore replaces the entire database with data from an uploaded JSON backup file.
func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims := auth.GetClaimsFromContext(r.Context())
	if claims == nil || claims.Role != "admin" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 10 MB limit
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("backup_file")
	if err != nil {
		http.Error(w, "Missing backup_file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var doc BackupDocument
	if err := json.NewDecoder(file).Decode(&doc); err != nil {
		http.Error(w, "Invalid JSON in backup file", http.StatusBadRequest)
		return
	}

	// Validate
	if doc.Metadata.Version != 1 {
		http.Error(w, fmt.Sprintf("Unsupported backup version: %d", doc.Metadata.Version), http.StatusBadRequest)
		return
	}
	if len(doc.Data.Roles) == 0 {
		http.Error(w, "Backup must contain at least one role", http.StatusBadRequest)
		return
	}
	if len(doc.Data.Users) == 0 {
		http.Error(w, "Backup must contain at least one user", http.StatusBadRequest)
		return
	}

	// Convert DTOs to DB models
	roles, err := dtosToRoles(doc.Data.Roles)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid role data: %v", err), http.StatusBadRequest)
		return
	}

	users, err := dtosToUsers(doc.Data.Users)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid user data: %v", err), http.StatusBadRequest)
		return
	}

	apiKeys, err := dtosToAPIKeys(doc.Data.APIKeys)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid API key data: %v", err), http.StatusBadRequest)
		return
	}

	badges := dtosToBadges(doc.Data.Badges)

	// Perform transactional restore
	if err := h.db.RestoreAll(roles, users, apiKeys, badges); err != nil {
		h.logger.Error("backup: restore failed", zap.Error(err))
		http.Error(w, "Restore failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clear cache so pages reflect restored data
	h.cache.Clear()

	h.logger.Info("backup: restore completed",
		zap.Int("roles", len(roles)),
		zap.Int("users", len(users)),
		zap.Int("api_keys", len(apiKeys)),
		zap.Int("badges", len(badges)),
		zap.String("restored_by", claims.Username),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "restored",
		"counts": map[string]int{
			"roles":    len(roles),
			"users":    len(users),
			"api_keys": len(apiKeys),
			"badges":   len(badges),
		},
	})
}
