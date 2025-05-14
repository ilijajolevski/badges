package database

import (
	"database/sql"
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestDatabaseOperations(t *testing.T) {
	// Create a test logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a temporary database file
	dbFile := "test_badges.db"
	defer os.Remove(dbFile)

	// Create a new database connection
	db, err := New(dbFile, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test creating a badge
	testBadge := &Badge{
		CommitID:        "test123",
		Type:            "badge",
		Status:          "valid",
		Issuer:          "Test Issuer",
		IssueDate:       "2023-01-01",
		SoftwareName:    "TestApp",
		SoftwareVersion: "v1.0.0",
		Notes:           sql.NullString{String: "Test notes", Valid: true},
		ExpiryDate:      sql.NullString{String: "2024-01-01", Valid: true},
		IssuerURL:       sql.NullString{String: "https://example.com", Valid: true},
	}

	// Create the badge
	err = db.CreateBadge(testBadge)
	if err != nil {
		t.Fatalf("Failed to create badge: %v", err)
	}

	// Test retrieving the badge
	retrievedBadge, err := db.GetBadge("test123")
	if err != nil {
		t.Fatalf("Failed to retrieve badge: %v", err)
	}

	if retrievedBadge == nil {
		t.Fatal("Retrieved badge is nil")
	}

	// Verify badge data
	if retrievedBadge.CommitID != testBadge.CommitID {
		t.Errorf("Expected CommitID %s, got %s", testBadge.CommitID, retrievedBadge.CommitID)
	}

	if retrievedBadge.Type != testBadge.Type {
		t.Errorf("Expected Type %s, got %s", testBadge.Type, retrievedBadge.Type)
	}

	if retrievedBadge.Status != testBadge.Status {
		t.Errorf("Expected Status %s, got %s", testBadge.Status, retrievedBadge.Status)
	}

	if retrievedBadge.Issuer != testBadge.Issuer {
		t.Errorf("Expected Issuer %s, got %s", testBadge.Issuer, retrievedBadge.Issuer)
	}

	if retrievedBadge.SoftwareName != testBadge.SoftwareName {
		t.Errorf("Expected SoftwareName %s, got %s", testBadge.SoftwareName, retrievedBadge.SoftwareName)
	}

	if retrievedBadge.SoftwareVersion != testBadge.SoftwareVersion {
		t.Errorf("Expected SoftwareVersion %s, got %s", testBadge.SoftwareVersion, retrievedBadge.SoftwareVersion)
	}

	if retrievedBadge.Notes.String != testBadge.Notes.String {
		t.Errorf("Expected Notes %s, got %s", testBadge.Notes.String, retrievedBadge.Notes.String)
	}

	if retrievedBadge.ExpiryDate.String != testBadge.ExpiryDate.String {
		t.Errorf("Expected ExpiryDate %s, got %s", testBadge.ExpiryDate.String, retrievedBadge.ExpiryDate.String)
	}

	if retrievedBadge.IssuerURL.String != testBadge.IssuerURL.String {
		t.Errorf("Expected IssuerURL %s, got %s", testBadge.IssuerURL.String, retrievedBadge.IssuerURL.String)
	}

	// Test updating the badge
	testBadge.Status = "expired"
	err = db.UpdateBadge(testBadge)
	if err != nil {
		t.Fatalf("Failed to update badge: %v", err)
	}

	// Retrieve the updated badge
	updatedBadge, err := db.GetBadge("test123")
	if err != nil {
		t.Fatalf("Failed to retrieve updated badge: %v", err)
	}

	if updatedBadge.Status != "expired" {
		t.Errorf("Expected Status %s, got %s", "expired", updatedBadge.Status)
	}

	// Test updating badge image
	imageData := []byte("test image data")
	err = db.UpdateBadgeImage("test123", "svg", imageData)
	if err != nil {
		t.Fatalf("Failed to update badge image: %v", err)
	}

	// Retrieve the badge with image
	badgeWithImage, err := db.GetBadge("test123")
	if err != nil {
		t.Fatalf("Failed to retrieve badge with image: %v", err)
	}

	if badgeWithImage.SVGContent.String != string(imageData) {
		t.Errorf("Expected SVG content %s, got %s", string(imageData), badgeWithImage.SVGContent.String)
	}

	// Test deleting the badge
	err = db.DeleteBadge("test123")
	if err != nil {
		t.Fatalf("Failed to delete badge: %v", err)
	}

	// Verify the badge is deleted
	deletedBadge, err := db.GetBadge("test123")
	if err != nil {
		t.Fatalf("Failed to check deleted badge: %v", err)
	}

	if deletedBadge != nil {
		t.Error("Badge was not deleted")
	}
}