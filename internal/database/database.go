package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
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
			software_url TEXT,
			notes TEXT,
			svg_content TEXT,
			expiry_date TEXT,
			issuer_url TEXT,
			custom_config TEXT,
			last_review TEXT,
			jpg_content BLOB,
			png_content BLOB,
			covered_version TEXT,
			repository_link TEXT,
			public_note TEXT,
			internal_note TEXT,
			contact_details TEXT,
			certificate_name TEXT,
			specialty_domain TEXT,
			software_sc_id TEXT,
			software_sc_url TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create badges table: %w", err)
	}

	// Create the roles table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			role_id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL,
			permissions TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create roles table: %w", err)
	}

	// Create the users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			role_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			last_login TIMESTAMP,
			status TEXT NOT NULL,
			failed_attempts INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (role_id) REFERENCES roles (role_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create the api_keys table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			api_key_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			api_key TEXT NOT NULL,
			name TEXT NOT NULL,
			permissions TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			last_used TIMESTAMP,
			status TEXT NOT NULL,
			ip_restrictions TEXT,
			FOREIGN KEY (user_id) REFERENCES users (user_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create api_keys table: %w", err)
	}

	// Add a test badge if it doesn't exist
	if err := addTestBadge(db); err != nil {
		return fmt.Errorf("failed to add test badge: %w", err)
	}

	// Add initial test badges if they don't exist
	if err := addInitialBadges(db); err != nil {
		return fmt.Errorf("failed to add initial test badges: %w", err)
	}

	// Add default admin role if it doesn't exist
	if err := addDefaultRole(db); err != nil {
		return fmt.Errorf("failed to add default admin role: %w", err)
	}

	// Add default admin user if no users exist
	if err := addDefaultAdminUser(db); err != nil {
		return fmt.Errorf("failed to add default admin user: %w", err)
	}

	return nil
}

// addDefaultRole adds a default admin role to the database if it doesn't already exist
func addDefaultRole(db *sql.DB) error {
	// Check if the admin role already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM roles WHERE name = ?", "admin").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for admin role: %w", err)
	}

	// If the role already exists, don't add it again
	if count > 0 {
		return nil
	}

	// Create admin role with full permissions
	permissions := `{
		"badges": {
			"read": true,
			"write": true,
			"delete": true
		},
		"users": {
			"read": true,
			"write": true,
			"delete": true
		},
		"api_keys": {
			"read": true,
			"write": true,
			"delete": true
		}
	}`

	// Generate a UUID for the role ID
	roleID := fmt.Sprintf("%x", time.Now().UnixNano())

	// Insert the admin role
	_, err = db.Exec(`
		INSERT INTO roles (
			role_id, name, description, permissions, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		roleID,                           // role_id
		"admin",                          // name
		"Administrator with full access", // description
		permissions,                      // permissions
		time.Now(),                       // created_at
		time.Now()) // updated_at
	if err != nil {
		return fmt.Errorf("failed to insert admin role: %w", err)
	}

	return nil
}

// addDefaultAdminUser adds a default admin user to the database if no users exist
func addDefaultAdminUser(db *sql.DB) error {
	// Check if any users exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for existing users: %w", err)
	}

	// If users already exist, don't add the default admin
	if count > 0 {
		return nil
	}

	// Get the admin role ID
	var roleID string
	err = db.QueryRow("SELECT role_id FROM roles WHERE name = ?", "admin").Scan(&roleID)
	if err != nil {
		return fmt.Errorf("failed to get admin role ID: %w", err)
	}

	// Generate a unique user ID
	userID := fmt.Sprintf("user_%x", time.Now().UnixNano())

	// Create a secure password
	password := "Admin@123" // Default password

	// Hash the password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12) // Cost factor of 12
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the admin user
	_, err = db.Exec(`
		INSERT INTO users (
			user_id, username, email, password_hash, first_name, last_name,
			role_id, created_at, updated_at, status, failed_attempts
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		userID,                 // user_id
		"admin",                // username
		"admin@example.com",    // email
		string(hashedPassword), // password_hash
		"Admin",                // first_name
		"User",                 // last_name
		roleID,                 // role_id
		time.Now(),             // created_at
		time.Now(),             // updated_at
		"active",               // status
		0) // failed_attempts
	if err != nil {
		return fmt.Errorf("failed to insert admin user: %w", err)
	}

	fmt.Println("Created default admin user with username 'admin' and password 'Admin@123'")
	return nil
}

// addTestBadge adds a test badge to the database if it doesn't already exist
func addTestBadge(db *sql.DB) error {
	// Check if the test badge already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM badges WHERE commit_id = ?", "softcat").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for test badge: %w", err)
	}

	// If the badge already exists, don't add it again
	if count > 0 {
		return nil
	}

	// Create a test badge with proper handling of NULL values
	notes := sql.NullString{String: "", Valid: true}
	expiryDate := sql.NullString{String: time.Now().AddDate(1, 0, 0).Format("2006-01-02"), Valid: true}
	issuerURL := sql.NullString{String: "https://certificates.software.geant.org", Valid: true}
	softwareURL := sql.NullString{String: "https://sc.geant.org/ui/project/SOFTCAT", Valid: true}
	customConfig := sql.NullString{String: `{"color_left":"#003f5f","color_right":"#FFFFFF","style":"3d","text_color_right":"#333", "border_color":"#ffffff", "horizontal_bars_color":"#bbb", "top_label_color":"#bbb"}`, Valid: true}
	lastReview := sql.NullString{String: time.Now().Format("2006-01-02"), Valid: true}
	coveredVersion := sql.NullString{String: "1.12.0", Valid: true}
	repositoryLink := sql.NullString{String: "https://bitbucket.software.geant.org/scm/sc/softwarecataloguegit.git", Valid: true}
	publicNote := sql.NullString{String: "This certificate certifies compliance with Software Licence standards", Valid: true}
	internalNote := sql.NullString{String: "Internal review comments and notes", Valid: true}
	contactDetails := sql.NullString{String: "certificates.software.geant.org", Valid: true}
	certificateName := sql.NullString{String: "Self-Assessed Dependencies", Valid: true}
	specialtyDomain := sql.NullString{String: "Software Licencing", Valid: true}
	softwareSCID := sql.NullString{String: "SOFTCAT", Valid: true}
	softwareSCURL := sql.NullString{String: "https://sc.geant.org/ui/project/SOFTCAT", Valid: true}

	// Insert the test badge
	_, err = db.Exec(`
		INSERT INTO badges (
			commit_id, type, status, issuer, issue_date, 
			software_name, software_version, software_url, notes, 
			expiry_date, issuer_url, custom_config, last_review,
			covered_version, repository_link, public_note, internal_note, contact_details,
			certificate_name, specialty_domain, software_sc_id, software_sc_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"softcat",                        // commit_id
		"badge",                          // type
		"valid",                          // status
		"GEANT WP9T2 Software Licencing", // issuer
		time.Now().Format("2006-01-02"),  // issue_date
		"GÉANT Software Catalogue",       // software_name
		"v1.12.0",                        // software_version
		softwareURL,                      // software_url
		notes,                            // notes
		expiryDate,                       // expiry_date
		issuerURL,                        // issuer_url
		customConfig,                     // custom_config
		lastReview,                       // last_review
		coveredVersion,                   // covered_version
		repositoryLink,                   // repository_link
		publicNote,                       // public_note
		internalNote,                     // internal_note
		contactDetails,                   // contact_details
		certificateName,                  // certificate_name
		specialtyDomain,                  // specialty_domain
		softwareSCID,                     // software_sc_id
		softwareSCURL) // software_sc_url
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
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content,
			covered_version, repository_link, public_note, internal_note, contact_details,
			certificate_name, specialty_domain, software_sc_id, software_sc_url
		FROM badges
		WHERE commit_id = ?
	`, commitID).Scan(
		&badge.CommitID, &badge.Type, &badge.Status, &badge.Issuer, &badge.IssueDate,
		&badge.SoftwareName, &badge.SoftwareVersion, &badge.SoftwareURL, &badge.Notes, &badge.SVGContent,
		&badge.ExpiryDate, &badge.IssuerURL, &badge.CustomConfig, &badge.LastReview, &badge.JPGContent, &badge.PNGContent,
		&badge.CoveredVersion, &badge.RepositoryLink, &badge.PublicNote, &badge.InternalNote, &badge.ContactDetails,
		&badge.CertificateName, &badge.SpecialtyDomain, &badge.SoftwareSCID, &badge.SoftwareSCURL,
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
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content,
			covered_version, repository_link, public_note, internal_note, contact_details,
			certificate_name, specialty_domain, software_sc_id, software_sc_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		badge.CommitID, badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.SoftwareURL, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.LastReview, badge.JPGContent, badge.PNGContent,
		badge.CoveredVersion, badge.RepositoryLink, badge.PublicNote, badge.InternalNote, badge.ContactDetails,
		badge.CertificateName, badge.SpecialtyDomain, badge.SoftwareSCID, badge.SoftwareSCURL,
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
			expiry_date = ?, issuer_url = ?, custom_config = ?, last_review = ?, jpg_content = ?, png_content = ?,
			covered_version = ?, repository_link = ?, public_note = ?, internal_note = ?, contact_details = ?,
			certificate_name = ?, specialty_domain = ?, software_sc_id = ?, software_sc_url = ?
		WHERE commit_id = ?
	`,
		badge.Type, badge.Status, badge.Issuer, badge.IssueDate,
		badge.SoftwareName, badge.SoftwareVersion, badge.SoftwareURL, badge.Notes, badge.SVGContent,
		badge.ExpiryDate, badge.IssuerURL, badge.CustomConfig, badge.LastReview, badge.JPGContent, badge.PNGContent,
		badge.CoveredVersion, badge.RepositoryLink, badge.PublicNote, badge.InternalNote, badge.ContactDetails,
		badge.CertificateName, badge.SpecialtyDomain, badge.SoftwareSCID, badge.SoftwareSCURL,
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
			expiry_date, issuer_url, custom_config, last_review, jpg_content, png_content,
			covered_version, repository_link, public_note, internal_note, contact_details,
			certificate_name, specialty_domain, software_sc_id, software_sc_url
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
			&badge.CoveredVersion, &badge.RepositoryLink, &badge.PublicNote, &badge.InternalNote, &badge.ContactDetails,
			&badge.CertificateName, &badge.SpecialtyDomain, &badge.SoftwareSCID, &badge.SoftwareSCURL,
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

// ==================== User CRUD Operations ====================

// CreateUser creates a new user in the database
func (db *DB) CreateUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (
			user_id, username, email, password_hash, first_name, last_name,
			role_id, created_at, updated_at, status, failed_attempts
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		user.UserID, user.Username, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.RoleID, user.CreatedAt, user.UpdatedAt, user.Status, user.FailedAttempts,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUser retrieves a user from the database by user ID
func (db *DB) GetUser(userID string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT 
			user_id, username, email, password_hash, first_name, last_name,
			role_id, created_at, updated_at, last_login, status, failed_attempts
		FROM users
		WHERE user_id = ?
	`, userID).Scan(
		&user.UserID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.RoleID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin, &user.Status, &user.FailedAttempts,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user from the database by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT 
			user_id, username, email, password_hash, first_name, last_name,
			role_id, created_at, updated_at, last_login, status, failed_attempts
		FROM users
		WHERE username = ?
	`, username).Scan(
		&user.UserID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.RoleID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin, &user.Status, &user.FailedAttempts,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user in the database
func (db *DB) UpdateUser(user *User) error {
	_, err := db.Exec(`
		UPDATE users SET
			username = ?, email = ?, password_hash = ?, first_name = ?, last_name = ?,
			role_id = ?, updated_at = ?, last_login = ?, status = ?, failed_attempts = ?
		WHERE user_id = ?
	`,
		user.Username, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.RoleID, user.UpdatedAt, user.LastLogin, user.Status, user.FailedAttempts,
		user.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user from the database
func (db *DB) DeleteUser(userID string) error {
	_, err := db.Exec("DELETE FROM users WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers retrieves all users from the database
func (db *DB) ListUsers() ([]*User, error) {
	rows, err := db.Query(`
		SELECT 
			user_id, username, email, password_hash, first_name, last_name,
			role_id, created_at, updated_at, last_login, status, failed_attempts
		FROM users
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.UserID, &user.Username, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
			&user.RoleID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin, &user.Status, &user.FailedAttempts,
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

// UpdateUserFailedAttempts increments the failed login attempts for a user
func (db *DB) UpdateUserFailedAttempts(userID string, attempts int) error {
	_, err := db.Exec("UPDATE users SET failed_attempts = ? WHERE user_id = ?", attempts, userID)
	if err != nil {
		return fmt.Errorf("failed to update user failed attempts: %w", err)
	}

	return nil
}

// UpdateUserLastLogin updates the last login timestamp for a user
func (db *DB) UpdateUserLastLogin(userID string, lastLogin time.Time) error {
	_, err := db.Exec("UPDATE users SET last_login = ? WHERE user_id = ?", lastLogin, userID)
	if err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}

	return nil
}

// ==================== Role CRUD Operations ====================

// CreateRole creates a new role in the database
func (db *DB) CreateRole(role *Role) error {
	_, err := db.Exec(`
		INSERT INTO roles (
			role_id, name, description, permissions, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		role.RoleID, role.Name, role.Description, role.Permissions, role.CreatedAt, role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetRole retrieves a role from the database by role ID
func (db *DB) GetRole(roleID string) (*Role, error) {
	var role Role
	err := db.QueryRow(`
		SELECT 
			role_id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE role_id = ?
	`, roleID).Scan(
		&role.RoleID, &role.Name, &role.Description, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Role not found
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// GetRoleByName retrieves a role from the database by name
func (db *DB) GetRoleByName(name string) (*Role, error) {
	var role Role
	err := db.QueryRow(`
		SELECT 
			role_id, name, description, permissions, created_at, updated_at
		FROM roles
		WHERE name = ?
	`, name).Scan(
		&role.RoleID, &role.Name, &role.Description, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Role not found
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}

	return &role, nil
}

// UpdateRole updates an existing role in the database
func (db *DB) UpdateRole(role *Role) error {
	_, err := db.Exec(`
		UPDATE roles SET
			name = ?, description = ?, permissions = ?, updated_at = ?
		WHERE role_id = ?
	`,
		role.Name, role.Description, role.Permissions, role.UpdatedAt,
		role.RoleID,
	)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeleteRole deletes a role from the database
func (db *DB) DeleteRole(roleID string) error {
	_, err := db.Exec("DELETE FROM roles WHERE role_id = ?", roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// ListRoles retrieves all roles from the database
func (db *DB) ListRoles() ([]*Role, error) {
	rows, err := db.Query(`
		SELECT 
			role_id, name, description, permissions, created_at, updated_at
		FROM roles
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var role Role
		err := rows.Scan(
			&role.RoleID, &role.Name, &role.Description, &role.Permissions, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, &role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}

// ==================== API Key CRUD Operations ====================

// CreateAPIKey creates a new API key in the database
func (db *DB) CreateAPIKey(apiKey *APIKey) error {
	_, err := db.Exec(`
		INSERT INTO api_keys (
			api_key_id, user_id, api_key, name, permissions,
			created_at, expires_at, status, ip_restrictions
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		apiKey.APIKeyID, apiKey.UserID, apiKey.APIKey, apiKey.Name, apiKey.Permissions,
		apiKey.CreatedAt, apiKey.ExpiresAt, apiKey.Status, apiKey.IPRestrictions,
	)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetAPIKey retrieves an API key from the database by API key ID
func (db *DB) GetAPIKey(apiKeyID string) (*APIKey, error) {
	var apiKey APIKey
	err := db.QueryRow(`
		SELECT 
			api_key_id, user_id, api_key, name, permissions,
			created_at, expires_at, last_used, status, ip_restrictions
		FROM api_keys
		WHERE api_key_id = ?
	`, apiKeyID).Scan(
		&apiKey.APIKeyID, &apiKey.UserID, &apiKey.APIKey, &apiKey.Name, &apiKey.Permissions,
		&apiKey.CreatedAt, &apiKey.ExpiresAt, &apiKey.LastUsed, &apiKey.Status, &apiKey.IPRestrictions,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // API key not found
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

// GetAPIKeyByKey retrieves an API key from the database by the hashed key value
func (db *DB) GetAPIKeyByKey(hashedKey string) (*APIKey, error) {
	var apiKey APIKey
	err := db.QueryRow(`
		SELECT 
			api_key_id, user_id, api_key, name, permissions,
			created_at, expires_at, last_used, status, ip_restrictions
		FROM api_keys
		WHERE api_key = ?
	`, hashedKey).Scan(
		&apiKey.APIKeyID, &apiKey.UserID, &apiKey.APIKey, &apiKey.Name, &apiKey.Permissions,
		&apiKey.CreatedAt, &apiKey.ExpiresAt, &apiKey.LastUsed, &apiKey.Status, &apiKey.IPRestrictions,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // API key not found
		}
		return nil, fmt.Errorf("failed to get API key by key: %w", err)
	}

	return &apiKey, nil
}

// UpdateAPIKey updates an existing API key in the database
func (db *DB) UpdateAPIKey(apiKey *APIKey) error {
	_, err := db.Exec(`
		UPDATE api_keys SET
			name = ?, permissions = ?, expires_at = ?, last_used = ?, status = ?, ip_restrictions = ?
		WHERE api_key_id = ?
	`,
		apiKey.Name, apiKey.Permissions, apiKey.ExpiresAt, apiKey.LastUsed, apiKey.Status, apiKey.IPRestrictions,
		apiKey.APIKeyID,
	)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}

// DeleteAPIKey deletes an API key from the database
func (db *DB) DeleteAPIKey(apiKeyID string) error {
	_, err := db.Exec("DELETE FROM api_keys WHERE api_key_id = ?", apiKeyID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// ListAPIKeys retrieves all API keys from the database
func (db *DB) ListAPIKeys() ([]*APIKey, error) {
	rows, err := db.Query(`
		SELECT 
			api_key_id, user_id, api_key, name, permissions,
			created_at, expires_at, last_used, status, ip_restrictions
		FROM api_keys
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*APIKey
	for rows.Next() {
		var apiKey APIKey
		err := rows.Scan(
			&apiKey.APIKeyID, &apiKey.UserID, &apiKey.APIKey, &apiKey.Name, &apiKey.Permissions,
			&apiKey.CreatedAt, &apiKey.ExpiresAt, &apiKey.LastUsed, &apiKey.Status, &apiKey.IPRestrictions,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, &apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return apiKeys, nil
}

// ListAPIKeysByUser retrieves all API keys for a specific user
func (db *DB) ListAPIKeysByUser(userID string) ([]*APIKey, error) {
	rows, err := db.Query(`
		SELECT 
			api_key_id, user_id, api_key, name, permissions,
			created_at, expires_at, last_used, status, ip_restrictions
		FROM api_keys
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys by user: %w", err)
	}
	defer rows.Close()

	var apiKeys []*APIKey
	for rows.Next() {
		var apiKey APIKey
		err := rows.Scan(
			&apiKey.APIKeyID, &apiKey.UserID, &apiKey.APIKey, &apiKey.Name, &apiKey.Permissions,
			&apiKey.CreatedAt, &apiKey.ExpiresAt, &apiKey.LastUsed, &apiKey.Status, &apiKey.IPRestrictions,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, &apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return apiKeys, nil
}

// UpdateAPIKeyLastUsed updates the last used timestamp for an API key
func (db *DB) UpdateAPIKeyLastUsed(apiKeyID string, lastUsed time.Time) error {
	_, err := db.Exec("UPDATE api_keys SET last_used = ? WHERE api_key_id = ?", lastUsed, apiKeyID)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// addInitialBadges ensures that specific initial badges exist in the database; if not, it inserts them
// using full field data loaded from a JSON file at db/initial_badges.json.
func addInitialBadges(db *sql.DB) error {
	// IDs we must ensure exist
	initialIDs := []string{"NMAAS_slSAD", "NMAAS_slVD", "EDREP20_slSAD", "EDREP20_slVD"}

	// Helper to build nullable strings
	ns := func(s string) sql.NullString {
		if s == "" {
			return sql.NullString{Valid: false}
		}
		return sql.NullString{String: s, Valid: true}
	}

	// Load JSON data
	type badgeJSON struct {
		CommitID        string `json:"commit_id"`
		Type            string `json:"type"`
		Status          string `json:"status"`
		Issuer          string `json:"issuer"`
		IssueDate       string `json:"issue_date"`
		SoftwareName    string `json:"software_name"`
		SoftwareVersion string `json:"software_version"`
		SoftwareURL     string `json:"software_url"`
		Notes           string `json:"notes"`
		ExpiryDate      string `json:"expiry_date"`
		IssuerURL       string `json:"issuer_url"`
		CustomConfig    string `json:"custom_config"`
		LastReview      string `json:"last_review"`
		CoveredVersion  string `json:"covered_version"`
		RepositoryLink  string `json:"repository_link"`
		PublicNote      string `json:"public_note"`
		InternalNote    string `json:"internal_note"`
		ContactDetails  string `json:"contact_details"`
		CertificateName string `json:"certificate_name"`
		SpecialtyDomain string `json:"specialty_domain"`
		SoftwareSCID    string `json:"software_sc_id"`
		SoftwareSCURL   string `json:"software_sc_url"`
	}

	// JSON is an object mapping commit_id -> badgeJSON
	var badgeMap map[string]badgeJSON
	jsonBytes, err := os.ReadFile("db/initial_badges.json")
	if err == nil {
		_ = json.Unmarshal(jsonBytes, &badgeMap)
	}

	for _, id := range initialIDs {
		// Check if the badge already exists
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM badges WHERE commit_id = ?", id).Scan(&count); err != nil {
			return fmt.Errorf("failed to check for initial badge %s: %w", id, err)
		}
		if count > 0 {
			continue
		}

		// Choose data source: JSON entry if present, otherwise sensible defaults
		var bj badgeJSON
		if badgeMap != nil {
			if v, ok := badgeMap[id]; ok {
				bj = v
			}
		}

		if bj.CommitID == "" {
			bj.CommitID = id
		}
		if bj.Type == "" {
			bj.Type = "badge"
		}
		if bj.Status == "" {
			bj.Status = "valid"
		}
		if bj.Issuer == "" {
			bj.Issuer = "GEANT WP9T2 Software Licencing"
		}
		if bj.IssueDate == "" {
			bj.IssueDate = time.Now().Format("2006-01-02")
		}
		if bj.SoftwareName == "" {
			bj.SoftwareName = "GÉANT Software Catalogue"
		}
		if bj.SoftwareVersion == "" {
			bj.SoftwareVersion = "v1.0.0"
		}

		// Insert all metadata fields (excluding media blobs which are generated elsewhere)
		_, err := db.Exec(`
			INSERT INTO badges (
				commit_id, type, status, issuer, issue_date,
				software_name, software_version, software_url, notes,
				expiry_date, issuer_url, custom_config, last_review,
				covered_version, repository_link, public_note, internal_note, contact_details,
				certificate_name, specialty_domain, software_sc_id, software_sc_url
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			bj.CommitID,
			bj.Type,
			bj.Status,
			bj.Issuer,
			bj.IssueDate,
			bj.SoftwareName,
			bj.SoftwareVersion,
			ns(bj.SoftwareURL),
			ns(bj.Notes),
			ns(bj.ExpiryDate),
			ns(bj.IssuerURL),
			ns(bj.CustomConfig),
			ns(bj.LastReview),
			ns(bj.CoveredVersion),
			ns(bj.RepositoryLink),
			ns(bj.PublicNote),
			ns(bj.InternalNote),
			ns(bj.ContactDetails),
			ns(bj.CertificateName),
			ns(bj.SpecialtyDomain),
			ns(bj.SoftwareSCID),
			ns(bj.SoftwareSCURL),
		)
		if err != nil {
			return fmt.Errorf("failed to insert initial badge %s: %w", id, err)
		}
	}

	return nil
}
