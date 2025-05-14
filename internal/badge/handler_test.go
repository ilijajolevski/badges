package badge

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

func TestBadgeHandler(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a temporary database file
	dbFile := "test_badges.db"
	defer os.Remove(dbFile)

	// Create a new database connection
	db, err := database.New(dbFile, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create a test badge in the database
	testBadge := &database.Badge{
		CommitID:        "test123",
		Type:            "badge",
		Status:          "valid",
		Issuer:          "Test Issuer",
		IssueDate:       "2023-01-01",
		SoftwareName:    "TestApp",
		SoftwareVersion: "v1.0.0",
		Notes:           sql.NullString{String: "Test notes", Valid: true},
	}

	err = db.CreateBadge(testBadge)
	if err != nil {
		t.Fatalf("Failed to create test badge: %v", err)
	}

	// Create a cache
	c := cache.New()

	// Create a badge handler
	handler := NewHandler(db, logger, c)

	// Test cases
	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "Valid badge request",
			url:            "/badge/test123",
			expectedStatus: http.StatusOK,
			expectedType:   "image/svg+xml",
		},
		{
			name:           "Valid badge request with format",
			url:            "/badge/test123?format=svg",
			expectedStatus: http.StatusOK,
			expectedType:   "image/svg+xml",
		},
		{
			name:           "Invalid badge request",
			url:            "/badge/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedType:   "text/plain; charset=utf-8",
		},
		{
			name:           "Invalid format",
			url:            "/badge/test123?format=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedType:   "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Check the content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Handler returned wrong content type: got %v want %v", contentType, tt.expectedType)
			}

			// For successful requests, check that the response body is not empty
			if tt.expectedStatus == http.StatusOK {
				if rr.Body.Len() == 0 {
					t.Error("Handler returned empty body")
				}
			}
		})
	}
}