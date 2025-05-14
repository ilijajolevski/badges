package main

import (
	"fmt"
	"log"
	"os"

	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

func main() {
	// Create a logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Set up a temporary database path for testing
	dbPath := "./db/test_badges.db"

	// Remove the database file if it exists
	os.Remove(dbPath)

	// Initialize the database
	db, err := database.New(dbPath, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Retrieve the test badge
	badge, err := db.GetBadge("test123")
	if err != nil {
		logger.Fatal("Failed to get test badge", zap.Error(err))
	}

	if badge == nil {
		logger.Fatal("Test badge not found")
	}

	// Print the test badge details
	fmt.Println("Test badge found:")
	fmt.Printf("  CommitID: %s\n", badge.CommitID)
	fmt.Printf("  Type: %s\n", badge.Type)
	fmt.Printf("  Status: %s\n", badge.Status)
	fmt.Printf("  Issuer: %s\n", badge.Issuer)
	fmt.Printf("  Issue Date: %s\n", badge.IssueDate)
	fmt.Printf("  Software: %s %s\n", badge.SoftwareName, badge.SoftwareVersion)
	
	if badge.Notes.Valid {
		fmt.Printf("  Notes: %s\n", badge.Notes.String)
	}
	
	if badge.ExpiryDate.Valid {
		fmt.Printf("  Expiry Date: %s\n", badge.ExpiryDate.String)
	}
	
	if badge.IssuerURL.Valid {
		fmt.Printf("  Issuer URL: %s\n", badge.IssuerURL.String)
	}
	
	if badge.CustomConfig.Valid {
		fmt.Printf("  Custom Config: %s\n", badge.CustomConfig.String)
	}

	logger.Info("Database initialization test completed successfully")
}