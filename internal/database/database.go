package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// DB represents a database connection
type DB struct {
	*sql.DB
	logger *zap.Logger
}

// New creates a new database connection
func New(dbPath string, logger *zap.Logger) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Check the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize the database
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &DB{DB: db, logger: logger}, nil
}

// initDB initializes the database schema
func initDB(db *sql.DB) error {
	// Create the badges table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS badges (
			commit_id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			issuer TEXT NOT NULL,
			issue_date TEXT NOT NULL,
			software_name TEXT NOT NULL,
			software_version TEXT NOT NULL,
			notes TEXT,
			svg_content TEXT,
			expiry_date TEXT,
			issuer_url TEXT,
			custom_config TEXT,
			jpg_content BLOB,
			png_content BLOB
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create badges table: %w", err)
	}

	// Add a test badge if it doesn't exist
	if err := addTestBadge(db); err != nil {
		return fmt.Errorf("failed to add test badge: %w", err)
	}

	return nil
}

// addTestBadge adds a test badge to the database if it doesn't already exist
func addTestBadge(db *sql.DB) error {
	// Check if the test badge already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM badges WHERE commit_id = ?", "test123").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for test badge: %w", err)
	}

	// If the badge already exists, don't add it again
	if count > 0 {
		return nil
	}

	// Create a test badge with proper handling of NULL values
	notes := sql.NullString{String: "This is a test badge for demonstration", Valid: true}
	expiryDate := sql.NullString{String: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), Valid: true}
	issuerURL := sql.NullString{String: "https://finki.edu.mk", Valid: true}
	customConfig := sql.NullString{String: `{"color_left":"#4B6CB7","color_right":"#182848","style":"3d"}`, Valid: true}

	// Insert the test badge
	_, err = db.Exec(`
		INSERT INTO badges (
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, notes, 
			expiry_date, issuer_url, custom_config
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"test123",                       // commit_id
		"badge",                         // type
		"valid",                         // status
		"FINKI Certification Board",     // issuer
		time.Now().Format("2006-01-02"), // issue_date
		"Badge Service",                 // software_name
		"v1.0.0",                        // software_version
		notes,                           // notes
		expiryDate,                      // expiry_date
		issuerURL,                       // issuer_url
		customConfig)                    // custom_config
	if err != nil {
		return fmt.Errorf("failed to insert test badge: %w", err)
	}

	return nil
}

// GetBadge retrieves a badge from the database by commit ID
func (db *DB) GetBadge(commitID string) (*Badge, error) {
	var badge Badge
	err := db.QueryRow(`
		SELECT 
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, notes, svg_content, 
			expiry_date, issuer_url, custom_config, jpg_content, png_content
		FROM badges
		WHERE commit_id = ?
	`, commitID).Scan(
		&badge.CommitID, &badge.Type, &badge.Status, &badge.Issuer, &badge.IssueDate,
		&badge.SoftwareName, &badge.SoftwareVersion, &badge.Notes, &badge.SVGContent,
		&badge.ExpiryDate, &badge.IssuerURL, &badge.CustomConfig, &badge.JPGContent, &badge.PNGContent,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Badge not found
		}
		return nil, fmt.Errorf("failed to get badge: %w", err)
	}

	return &badge, nil
}

// CreateBadge creates a new badge in the database
func (db *DB) CreateBadge(badge *Badge) error {
	_, err := db.Exec(`
		INSERT INTO badges (
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, notes, svg_content, 
			expiry_date, issuer_url, custom_config, jpg_content, png_content
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		badge.CommitID, badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.JPGContent, badge.PNGContent,
	)
	if err != nil {
		return fmt.Errorf("failed to create badge: %w", err)
	}

	return nil
}

// UpdateBadge updates an existing badge in the database
func (db *DB) UpdateBadge(badge *Badge) error {
	_, err := db.Exec(`
		UPDATE badges SET
			type = ?, status = ?, issuer = ?, issue_date = ?,
			software_name = ?, software_version = ?, notes = ?, svg_content = ?,
			expiry_date = ?, issuer_url = ?, custom_config = ?, jpg_content = ?, png_content = ?
		WHERE commit_id = ?
	`,
		badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.JPGContent, badge.PNGContent,
		badge.CommitID,
	)
	if err != nil {
		return fmt.Errorf("failed to update badge: %w", err)
	}

	return nil
}

// DeleteBadge deletes a badge from the database
func (db *DB) DeleteBadge(commitID string) error {
	_, err := db.Exec("DELETE FROM badges WHERE commit_id = ?", commitID)
	if err != nil {
		return fmt.Errorf("failed to delete badge: %w", err)
	}

	return nil
}

// UpdateBadgeImage updates the image content of a badge
func (db *DB) UpdateBadgeImage(commitID, format string, content []byte) error {
	var query string
	switch format {
	case "svg":
		query = "UPDATE badges SET svg_content = ? WHERE commit_id = ?"
	case "jpg":
		query = "UPDATE badges SET jpg_content = ? WHERE commit_id = ?"
	case "png":
		query = "UPDATE badges SET png_content = ? WHERE commit_id = ?"
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	_, err := db.Exec(query, content, commitID)
	if err != nil {
		return fmt.Errorf("failed to update badge image: %w", err)
	}

	return nil
}

// ListBadges retrieves all badges from the database
func (db *DB) ListBadges() ([]*Badge, error) {
	rows, err := db.Query(`
		SELECT 
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, notes, svg_content, 
			expiry_date, issuer_url, custom_config, jpg_content, png_content
		FROM badges
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list badges: %w", err)
	}
	defer rows.Close()

	var badges []*Badge
	for rows.Next() {
		var badge Badge
		err := rows.Scan(
			&badge.CommitID, &badge.Type, &badge.Status, &badge.Issuer, &badge.IssueDate,
			&badge.SoftwareName, &badge.SoftwareVersion, &badge.Notes, &badge.SVGContent,
			&badge.ExpiryDate, &badge.IssuerURL, &badge.CustomConfig, &badge.JPGContent, &badge.PNGContent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan badge: %w", err)
		}
		badges = append(badges, &badge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating badges: %w", err)
	}

	return badges, nil
}
