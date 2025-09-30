package middleware

import (
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ErrorTemplateData represents the data passed to the error template
type ErrorTemplateData struct {
	StatusCode  int
	Title       string
	Message     string
	CurrentYear int
}

// ErrorHandler is a middleware that handles errors
type ErrorHandler struct {
	logger   *zap.Logger
	template *template.Template
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *zap.Logger) (*ErrorHandler, error) {
	// Parse the error template
	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		return nil, err
	}

	return &ErrorHandler{
		logger:   logger,
		template: tmpl,
	}, nil
}

// Middleware returns a middleware function that handles errors
func (h *ErrorHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a custom response writer to capture the status code
		crw := &customResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(crw, r)

		// If the status code is an error, render the error page
		if crw.statusCode >= 400 {
			h.renderErrorPage(crw.ResponseWriter, crw.statusCode, r)
		}
	})
}

// renderErrorPage renders the error page
func (h *ErrorHandler) renderErrorPage(w http.ResponseWriter, statusCode int, r *http.Request) {
	// Get the error title and message based on the status code
	title, message := getErrorDetails(statusCode)

	// Prepare template data
	data := ErrorTemplateData{
		StatusCode:  statusCode,
		Title:       title,
		Message:     message,
		CurrentYear: time.Now().Year(),
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render error template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getErrorDetails returns the title and message for an error status code
func getErrorDetails(statusCode int) (string, string) {
	switch statusCode {
	case http.StatusBadRequest:
		return "Bad Request", "The request could not be understood by the server due to malformed syntax."
	case http.StatusNotFound:
		return "Not Found", "The requested resource could not be found on this server."
	case http.StatusInternalServerError:
		return "Internal Server Error", "The server encountered an unexpected condition which prevented it from fulfilling the request."
	case http.StatusForbidden:
		return "Forbidden", "You don't have permission to access this resource."
	case http.StatusTooManyRequests:
		return "Too Many Requests", "You have sent too many requests in a given amount of time."
	default:
		return "Error", "An error occurred while processing your request."
	}
}

// customResponseWriter is a custom response writer that captures the status code
type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
	wroteHeader bool
}

// WriteHeader captures the status code
func (crw *customResponseWriter) WriteHeader(statusCode int) {
	crw.statusCode = statusCode
	crw.wroteHeader = true
	crw.ResponseWriter.WriteHeader(statusCode)
}

// Write writes the data to the connection as part of an HTTP reply
func (crw *customResponseWriter) Write(b []byte) (int, error) {
	if !crw.wroteHeader {
		crw.WriteHeader(http.StatusOK)
	}

	// Check if this is an SVG or other XML content
	contentType := crw.ResponseWriter.Header().Get("Content-Type")
	if contentType == "image/svg+xml" || strings.HasPrefix(contentType, "application/xml") || strings.HasPrefix(contentType, "text/xml") {
		// For SVG/XML content, write directly without HTML escaping
		return crw.ResponseWriter.Write(b)
	}

	return crw.ResponseWriter.Write(b)
}

// Sanitizer is a middleware that sanitizes input
type Sanitizer struct {
	logger *zap.Logger
}

// NewSanitizer creates a new sanitizer
func NewSanitizer(logger *zap.Logger) *Sanitizer {
	return &Sanitizer{
		logger: logger,
	}
}

// Middleware returns a middleware function that sanitizes input
func (s *Sanitizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate commit ID
		if r.URL.Path != "/" && r.URL.Path != "/badges" {
			// Extract commit ID from URL
			var commitID string
			pathLen := len(r.URL.Path)

			if pathLen >= 7 && r.URL.Path[:7] == "/badge/" {
				commitID = r.URL.Path[7:]
			} else if pathLen >= 13 && r.URL.Path[:13] == "/certificate/" {
				commitID = r.URL.Path[13:]
			} else if pathLen >= 9 && r.URL.Path[:9] == "/details/" {
				commitID = r.URL.Path[9:]
			}

			// Remove any trailing path segments
			if idx := regexp.MustCompile(`[/]`).FindStringIndex(commitID); idx != nil {
				commitID = commitID[:idx[0]]
			}

			// Validate commit ID format (alphanumeric and underscore, 6-40 chars)
			if commitID != "" && !regexp.MustCompile(`^[a-zA-Z0-9_]{6,40}$`).MatchString(commitID) {
				s.logger.Warn("Invalid commit ID format", zap.String("commit_id", commitID))
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// RateLimiter is a middleware that limits the rate of requests
type RateLimiter struct {
	logger    *zap.Logger
	requests  map[string][]time.Time
	mu        sync.Mutex
	limit     int
	window    time.Duration
	cleanupInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger *zap.Logger, limit int, window time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		logger:    logger,
		requests:  make(map[string][]time.Time),
		limit:     limit,
		window:    window,
		cleanupInterval: time.Minute,
	}

	// Start the cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Middleware returns a middleware function that limits the rate of requests
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the client IP
		clientIP := r.RemoteAddr

		// Check if the client has exceeded the rate limit
		if rl.isLimited(clientIP) {
			rl.logger.Warn("Rate limit exceeded", zap.String("client_ip", clientIP))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// isLimited checks if the client has exceeded the rate limit
func (rl *RateLimiter) isLimited(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get the client's requests
	clientRequests, exists := rl.requests[clientIP]
	if !exists {
		rl.requests[clientIP] = []time.Time{now}
		return false
	}

	// Filter out requests outside the window
	var recentRequests []time.Time
	for _, t := range clientRequests {
		if t.After(windowStart) {
			recentRequests = append(recentRequests, t)
		}
	}

	// Check if the client has exceeded the limit
	if len(recentRequests) >= rl.limit {
		return true
	}

	// Add the current request
	rl.requests[clientIP] = append(recentRequests, now)
	return false
}

// cleanup periodically removes expired entries from the requests map
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for clientIP, requests := range rl.requests {
			var recentRequests []time.Time
			for _, t := range requests {
				if t.After(windowStart) {
					recentRequests = append(recentRequests, t)
				}
			}

			if len(recentRequests) == 0 {
				delete(rl.requests, clientIP)
			} else {
				rl.requests[clientIP] = recentRequests
			}
		}

		rl.mu.Unlock()
	}
}

// RequestLogger is a middleware that logs HTTP requests
type RequestLogger struct {
	logger *zap.Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger *zap.Logger) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// Middleware returns a middleware function that logs HTTP requests
func (rl *RequestLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a custom response writer to capture the status code
		crw := &customResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Get the start time
		startTime := time.Now()

		// Call the next handler
		next.ServeHTTP(crw, r)

		// Calculate the request duration
		duration := time.Since(startTime)

		// Log the request at trace level (using Debug)
		rl.logger.Debug("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("client_ip", r.RemoteAddr),
			zap.Int("status", crw.statusCode),
			zap.Duration("duration", duration),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}
