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

// GetUserByEmail retrieves a user from the database by email
func (db *DB) GetUserByEmail(email string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT 
			user_id, email, name, is_superadmin, last_login, created_at
		FROM users
		WHERE email = ?
	`, email).Scan(
		&user.UserID, &user.Email, &user.Name, &user.IsSuperadmin, &user.LastLogin, &user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new user in the database
func (db *DB) CreateUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (
			user_id, email, name, is_superadmin, created_at
		) VALUES (?, ?, ?, ?, ?)
	`,
		user.UserID, user.Email, user.Name, user.IsSuperadmin, user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUserLastLogin updates the last login time for a user
func (db *DB) UpdateUserLastLogin(userID string) error {
	_, err := db.Exec(`
		UPDATE users
		SET last_login = ?
		WHERE user_id = ?
	`,
		time.Now().Format(time.RFC3339), userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}

	return nil
}

// AddUserToGroup adds a user to a domain authority group
func (db *DB) AddUserToGroup(userID string, groupID string) error {
	_, err := db.Exec(`
		INSERT INTO user_group_memberships (
			user_id, group_id, joined_at
		) VALUES (?, ?, ?)
	`,
		userID, groupID, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user from the database by ID
func (db *DB) GetUserByID(userID string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT 
			user_id, email, name, is_superadmin, last_login, created_at
		FROM users
		WHERE user_id = ?
	`, userID).Scan(
		&user.UserID, &user.Email, &user.Name, &user.IsSuperadmin, &user.LastLogin, &user.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// IsUserInGroup checks if a user is a member of a group
func (db *DB) IsUserInGroup(userID string, groupID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_group_memberships
		WHERE user_id = ? AND group_id = ?
	`, userID, groupID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user is in group: %w", err)
	}

	return count > 0, nil
}

// UserHasAccessToBadgeType checks if a user has access to a badge type
func (db *DB) UserHasAccessToBadgeType(userID string, badgeType string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM user_group_memberships ugm
		JOIN badge_group_associations bga ON ugm.group_id = bga.group_id
		WHERE ugm.user_id = ? AND bga.badge_type = ?
	`, userID, badgeType).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user has access to badge type: %w", err)
	}

	return count > 0, nil
}

// ListUsers retrieves all users from the database
func (db *DB) ListUsers() ([]*User, error) {
	rows, err := db.Query(`
		SELECT 
			user_id, email, name, is_superadmin, last_login, created_at
		FROM users
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.UserID, &user.Email, &user.Name, &user.IsSuperadmin, &user.LastLogin, &user.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// GetUserGroups retrieves all groups a user is a member of
func (db *DB) GetUserGroups(userID string) ([]*DomainAuthorityGroup, error) {
	rows, err := db.Query(`
		SELECT 
			dag.group_id, dag.name, dag.description, dag.created_at
		FROM domain_authority_groups dag
		JOIN user_group_memberships ugm ON dag.group_id = ugm.group_id
		WHERE ugm.user_id = ?
		ORDER BY dag.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}
	defer rows.Close()

	var groups []*DomainAuthorityGroup
	for rows.Next() {
		var group DomainAuthorityGroup
		err := rows.Scan(
			&group.GroupID, &group.Name, &group.Description, &group.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, &group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}

// RemoveUserFromGroup removes a user from a domain authority group
func (db *DB) RemoveUserFromGroup(userID string, groupID string) error {
	_, err := db.Exec(`
		DELETE FROM user_group_memberships
		WHERE user_id = ? AND group_id = ?
	`, userID, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}

	return nil
}

// ListGroups retrieves all domain authority groups from the database
func (db *DB) ListGroups() ([]*DomainAuthorityGroup, error) {
	rows, err := db.Query(`
		SELECT 
			group_id, name, description, created_at
		FROM domain_authority_groups
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []*DomainAuthorityGroup
	for rows.Next() {
		var group DomainAuthorityGroup
		err := rows.Scan(
			&group.GroupID, &group.Name, &group.Description, &group.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, &group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}

// CreateGroup creates a new domain authority group in the database
func (db *DB) CreateGroup(group *DomainAuthorityGroup) error {
	_, err := db.Exec(`
		INSERT INTO domain_authority_groups (
			group_id, name, description, created_at
		) VALUES (?, ?, ?, ?)
	`,
		group.GroupID, group.Name, group.Description, group.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	return nil
}

// GetGroupByID retrieves a domain authority group from the database by ID
func (db *DB) GetGroupByID(groupID string) (*DomainAuthorityGroup, error) {
	var group DomainAuthorityGroup
	err := db.QueryRow(`
		SELECT 
			group_id, name, description, created_at
		FROM domain_authority_groups
		WHERE group_id = ?
	`, groupID).Scan(
		&group.GroupID, &group.Name, &group.Description, &group.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Group not found
		}
		return nil, fmt.Errorf("failed to get group by ID: %w", err)
	}

	return &group, nil
}

// UpdateGroup updates a domain authority group in the database
func (db *DB) UpdateGroup(group *DomainAuthorityGroup) error {
	_, err := db.Exec(`
		UPDATE domain_authority_groups
		SET name = ?, description = ?
		WHERE group_id = ?
	`,
		group.Name, group.Description, group.GroupID,
	)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

// DeleteGroup deletes a domain authority group from the database
func (db *DB) DeleteGroup(groupID string) error {
	_, err := db.Exec(`
		DELETE FROM domain_authority_groups
		WHERE group_id = ?
	`, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

// GetGroupBadges retrieves all badge types assigned to a group
func (db *DB) GetGroupBadges(groupID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT badge_type
		FROM badge_group_associations
		WHERE group_id = ?
		ORDER BY badge_type
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group badges: %w", err)
	}
	defer rows.Close()

	var badgeTypes []string
	for rows.Next() {
		var badgeType string
		err := rows.Scan(&badgeType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan badge type: %w", err)
		}
		badgeTypes = append(badgeTypes, badgeType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating badge types: %w", err)
	}

	return badgeTypes, nil
}

// AssignBadgeToGroup assigns a badge type to a domain authority group
func (db *DB) AssignBadgeToGroup(badgeType string, groupID string) error {
	_, err := db.Exec(`
		INSERT INTO badge_group_associations (
			badge_type, group_id, assigned_at
		) VALUES (?, ?, ?)
	`,
		badgeType, groupID, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to assign badge to group: %w", err)
	}

	return nil
}

// RemoveBadgeFromGroup removes a badge type from a domain authority group
func (db *DB) RemoveBadgeFromGroup(badgeType string, groupID string) error {
	_, err := db.Exec(`
		DELETE FROM badge_group_associations
		WHERE badge_type = ? AND group_id = ?
	`, badgeType, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove badge from group: %w", err)
	}

	return nil
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
			software_url TEXT,
			notes TEXT,
			svg_content TEXT,
			expiry_date TEXT,
			issuer_url TEXT,
			custom_config TEXT,
			last_review TEXT,
			jpg_content BLOB,
			png_content BLOB
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create badges table: %w", err)
	}

	// Create the users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_superadmin BOOLEAN NOT NULL DEFAULT 0,
			last_login TEXT,
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create the domain authority groups table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS domain_authority_groups (
			group_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create domain authority groups table: %w", err)
	}

	// Create the user-group membership table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_group_memberships (
			user_id TEXT NOT NULL,
			group_id TEXT NOT NULL,
			joined_at TEXT NOT NULL,
			PRIMARY KEY (user_id, group_id),
			FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
			FOREIGN KEY (group_id) REFERENCES domain_authority_groups(group_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user-group membership table: %w", err)
	}

	// Create the badge-group association table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS badge_group_associations (
			badge_type TEXT NOT NULL,
			group_id TEXT NOT NULL,
			assigned_at TEXT NOT NULL,
			PRIMARY KEY (badge_type, group_id),
			FOREIGN KEY (group_id) REFERENCES domain_authority_groups(group_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create badge-group association table: %w", err)
	}

	// Add a test badge if it doesn't exist
	if err := addTestBadge(db); err != nil {
		return fmt.Errorf("failed to add test badge: %w", err)
	}

	// Create demo group and initial superadmin if they don't exist
	if err := initializeAuthData(db); err != nil {
		return fmt.Errorf("failed to initialize auth data: %w", err)
	}

	return nil
}

// initializeAuthData creates the demo group and initial superadmin if they don't exist
func initializeAuthData(db *sql.DB) error {
	// Create the demo group if it doesn't exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM domain_authority_groups WHERE group_id = ?", "demo_group").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for demo group: %w", err)
	}

	// If the demo group doesn't exist, create it
	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO domain_authority_groups (
				group_id, name, description, created_at
			) VALUES (?, ?, ?, ?)
		`,
			"demo_group",
			"Demo Badge Group",
			"Group for test badges and new users",
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to create demo group: %w", err)
		}
	}

	// Create the initial superadmin if it doesn't exist
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "badge-admin@gmail.com").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for initial superadmin: %w", err)
	}

	// If the initial superadmin doesn't exist, create it
	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO users (
				user_id, email, name, is_superadmin, created_at
			) VALUES (?, ?, ?, ?, ?)
		`,
			"admin123",
			"badge-admin@gmail.com",
			"Badge Admin",
			true,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to create initial superadmin: %w", err)
		}

		// Add the initial superadmin to the demo group
		_, err = db.Exec(`
			INSERT INTO user_group_memberships (
				user_id, group_id, joined_at
			) VALUES (?, ?, ?)
		`,
			"admin123",
			"demo_group",
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to add initial superadmin to demo group: %w", err)
		}
	}

	// Assign test badges to the demo group
	// First, check if the test badge is already assigned to the demo group
	err = db.QueryRow("SELECT COUNT(*) FROM badge_group_associations WHERE badge_type = ? AND group_id = ?", "badge", "demo_group").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for badge-group association: %w", err)
	}

	// If the test badge is not assigned to the demo group, assign it
	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO badge_group_associations (
				badge_type, group_id, assigned_at
			) VALUES (?, ?, ?)
		`,
			"badge",
			"demo_group",
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to assign test badge to demo group: %w", err)
		}
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
	softwareURL := sql.NullString{String: "https://github.com/finki/badges", Valid: true}
	customConfig := sql.NullString{String: `{"color_left":"#4B6CB7","color_right":"#182848","style":"3d"}`, Valid: true}
	lastReview := sql.NullString{String: time.Now().Format("2006-01-02"), Valid: true}

	// Insert the test badge
	_, err = db.Exec(`
		INSERT INTO badges (
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, software_url, notes, 
			expiry_date, issuer_url, custom_config, last_review
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"test123",                       // commit_id
		"badge",                         // type
		"valid",                         // status
		"FINKI Certification Board",     // issuer
		time.Now().Format("2006-01-02"), // issue_date
		"Badge Service",                 // software_name
		"v1.0.0",                        // software_version
		softwareURL,                     // software_url
		notes,                           // notes
		expiryDate,                      // expiry_date
		issuerURL,                       // issuer_url
		customConfig,                    // custom_config
		lastReview)                      // last_review
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
			software_name, software_version, software_url, notes, svg_content, 
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content
		FROM badges
		WHERE commit_id = ?
	`, commitID).Scan(
		&badge.CommitID, &badge.Type, &badge.Status, &badge.Issuer, &badge.IssueDate,
		&badge.SoftwareName, &badge.SoftwareVersion, &badge.SoftwareURL, &badge.Notes, &badge.SVGContent,
		&badge.ExpiryDate, &badge.IssuerURL, &badge.CustomConfig, &badge.LastReview, &badge.JPGContent, &badge.PNGContent,
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
			software_name, software_version, software_url, notes, svg_content, 
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		badge.CommitID, badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.SoftwareURL, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.LastReview, badge.JPGContent, badge.PNGContent,
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
			software_name = ?, software_version = ?, software_url = ?, notes = ?, svg_content = ?,
			expiry_date = ?, issuer_url = ?, custom_config = ?, last_review = ?, jpg_content = ?, png_content = ?
		WHERE commit_id = ?
	`,
		badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.SoftwareURL, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.LastReview, badge.JPGContent, badge.PNGContent,
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
			software_name, software_version, software_url, notes, svg_content, 
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content
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
			&badge.SoftwareName, &badge.SoftwareVersion, &badge.SoftwareURL, &badge.Notes, &badge.SVGContent,
			&badge.ExpiryDate, &badge.IssuerURL, &badge.CustomConfig, &badge.LastReview, &badge.JPGContent, &badge.PNGContent,
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
