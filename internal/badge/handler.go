package badge

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/certificate"
	"github.com/finki/badges/internal/database"
	"github.com/finki/badges/pkg/utils"
	"go.uber.org/zap"
)

// Handler handles badge requests
type Handler struct {
	db                 *database.DB
	logger             *zap.Logger
	cache              *cache.Cache
	badgeGenerator     *Generator
	certificateGenerator *certificate.Generator
}

// NewHandler creates a new badge handler
func NewHandler(db *database.DB, logger *zap.Logger, cache *cache.Cache) *Handler {
	return &Handler{
		db:                 db,
		logger:             logger,
		cache:              cache,
		badgeGenerator:     NewGenerator(),
		certificateGenerator: certificate.NewGenerator(),
	}
}

// ServeHTTP handles HTTP requests for badges
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract commit ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/badge/")
	commitID := strings.Split(path, "/")[0]

	if commitID == "" {
		http.Error(w, "Missing commit ID", http.StatusBadRequest)
		return
	}

	// Get format from query parameter (default: svg)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "svg"
	}

	// Validate format
	if format != "svg" && format != "png" && format != "jpg" {
		http.Error(w, "Invalid format. Supported formats: svg, png, jpg", http.StatusBadRequest)
		return
	}

	// Get outlook from query parameter (default: badge)
	outlook := r.URL.Query().Get("outlook")
	if outlook == "" {
		outlook = "badge"
	}

	// Validate outlook
	if outlook != "badge" && outlook != "certificate" {
		http.Error(w, "Invalid outlook. Supported outlooks: badge, certificate", http.StatusBadRequest)
		return
	}

	// Check for no_cache parameter
	noCache := r.URL.Query().Get("no_cache") == "true"

	// Try to get from cache first (unless no_cache is true)
	cacheKey := fmt.Sprintf("badge:%s:%s:%s", commitID, format, r.URL.RawQuery)
	if !noCache {
		if cachedData, found := h.cache.Get(cacheKey); found {
			h.serveImage(w, cachedData, format)
			return
		}
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

	// Apply query parameters to badge configuration
	if err := h.applyQueryParams(badge, r); err != nil {
		h.logger.Error("Failed to apply query parameters", zap.Error(err))
		http.Error(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	// Generate or retrieve image based on format and outlook
	var imageData []byte
	var genErr error

	// Choose the appropriate generator based on outlook
	var generator interface {
		GenerateSVG(badge *database.Badge) ([]byte, error)
	}

	if outlook == "badge" {
		generator = h.badgeGenerator
	} else {
		generator = h.certificateGenerator
	}

	switch format {
	case "svg":
		// Generate SVG
		imageData, genErr = generator.GenerateSVG(badge)
	case "png":
		// Check if PNG is already in the database
		if badge.PNGContent != nil {
			imageData = badge.PNGContent
		} else {
			// Generate SVG first
			svgData, err := generator.GenerateSVG(badge)
			if err != nil {
				h.logger.Error("Failed to generate SVG", zap.Error(err))
				http.Error(w, "Failed to generate image", http.StatusInternalServerError)
				return
			}

			// Convert SVG to PNG
			imageData, genErr = utils.SVGToPNG(svgData, 0, 0)
			if genErr == nil {
				// Store PNG in database for future use
				if err := h.db.UpdateBadgeImage(commitID, "png", imageData); err != nil {
					h.logger.Error("Failed to update PNG in database", zap.Error(err))
				}
			}
		}
	case "jpg":
		// Check if JPG is already in the database
		if badge.JPGContent != nil {
			imageData = badge.JPGContent
		} else {
			// Generate SVG first
			svgData, err := generator.GenerateSVG(badge)
			if err != nil {
				h.logger.Error("Failed to generate SVG", zap.Error(err))
				http.Error(w, "Failed to generate image", http.StatusInternalServerError)
				return
			}

			// Convert SVG to JPG
			imageData, genErr = utils.SVGToJPG(svgData, 0, 0)
			if genErr == nil {
				// Store JPG in database for future use
				if err := h.db.UpdateBadgeImage(commitID, "jpg", imageData); err != nil {
					h.logger.Error("Failed to update JPG in database", zap.Error(err))
				}
			}
		}
	}

	if genErr != nil {
		h.logger.Error("Failed to generate image", zap.Error(genErr), zap.String("format", format))
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}

	// Cache the result
	h.cache.Set(cacheKey, imageData, 5*time.Minute)

	// Serve the image
	h.serveImage(w, imageData, format)
}

// serveImage serves an image with the appropriate content type
func (h *Handler) serveImage(w http.ResponseWriter, data []byte, format string) {
	switch format {
	case "svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case "png":
		w.Header().Set("Content-Type", "image/png")
	case "jpg":
		w.Header().Set("Content-Type", "image/jpeg")
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write(data)
}

// applyQueryParams applies query parameters to the badge configuration
func (h *Handler) applyQueryParams(badge *database.Badge, r *http.Request) error {
	// Get current custom config or create a new one
	config, err := badge.GetCustomConfig()
	if err != nil {
		return err
	}

	// Apply query parameters
	if colorLeft := r.URL.Query().Get("color_left"); colorLeft != "" {
		config.ColorLeft = colorLeft
	}

	if colorRight := r.URL.Query().Get("color_right"); colorRight != "" {
		config.ColorRight = colorRight
	}

	if textColor := r.URL.Query().Get("text_color"); textColor != "" {
		config.TextColor = textColor
	}

	if textColorLeft := r.URL.Query().Get("text_color_left"); textColorLeft != "" {
		config.TextColorLeft = textColorLeft
	}

	if textColorRight := r.URL.Query().Get("text_color_right"); textColorRight != "" {
		config.TextColorRight = textColorRight
	}

	if logo := r.URL.Query().Get("logo"); logo != "" {
		config.LogoURL = logo
	}

	if fontSize := r.URL.Query().Get("font_size"); fontSize != "" {
		var size int
		if _, err := fmt.Sscanf(fontSize, "%d", &size); err == nil && size >= 8 && size <= 16 {
			config.FontSize = size
		}
	}

	if style := r.URL.Query().Get("style"); style != "" {
		if style == "flat" || style == "3d" {
			config.Style = style
		}
	}

	// Update badge with new config
	return badge.SetCustomConfig(config)
}
