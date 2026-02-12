package backup

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/database"
	"go.uber.org/zap"
)

// setupTestDB creates a temporary SQLite database for testing.
func setupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()
	dbFile := "test_backup_" + t.Name() + ".db"
	logger, _ := zap.NewDevelopment()
	db, err := database.New(dbFile, logger)
	if err != nil {
		t.Fatalf("failed to create test DB: %v", err)
	}
	cleanup := func() {
		db.Close()
		os.Remove(dbFile)
	}
	return db, cleanup
}

// adminContext returns a context with admin JWT claims.
func adminContext() context.Context {
	claims := &auth.Claims{
		UserID:   "admin-user-id",
		Username: "admin",
		Email:    "admin@example.com",
		Role:     "admin",
	}
	claims.Permissions.Users.Write = true
	return auth.AddClaimsToContext(context.Background(), claims)
}

// nonAdminContext returns a context with non-admin JWT claims.
func nonAdminContext() context.Context {
	claims := &auth.Claims{
		UserID:   "user-id",
		Username: "viewer",
		Email:    "viewer@example.com",
		Role:     "viewer",
	}
	return auth.AddClaimsToContext(context.Background(), claims)
}

func TestBackupSuccess(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	req := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Backup(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	if cd := resp.Header.Get("Content-Disposition"); cd == "" {
		t.Error("expected Content-Disposition header")
	}

	body, _ := io.ReadAll(resp.Body)
	var doc BackupDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("failed to unmarshal backup JSON: %v", err)
	}

	if doc.Metadata.Version != 1 {
		t.Errorf("expected version 1, got %d", doc.Metadata.Version)
	}
	if doc.Metadata.Application != "CertifyHub" {
		t.Errorf("expected application CertifyHub, got %s", doc.Metadata.Application)
	}
	if doc.Metadata.CreatedBy != "admin" {
		t.Errorf("expected created_by admin, got %s", doc.Metadata.CreatedBy)
	}
	if len(doc.Data.Roles) == 0 {
		t.Error("expected at least one role in backup")
	}
	if len(doc.Data.Users) == 0 {
		t.Error("expected at least one user in backup")
	}
	if len(doc.Data.Badges) == 0 {
		t.Error("expected at least one badge in backup")
	}
}

func TestBackupExcludesBinaryFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add binary content to a badge
	badges, _ := db.ListBadges()
	if len(badges) > 0 {
		db.UpdateBadgeImage(badges[0].CommitID, "png", []byte("fakepng"))
		db.UpdateBadgeImage(badges[0].CommitID, "jpg", []byte("fakejpg"))
	}

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	req := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Backup(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)

	// The JSON should not contain the binary field keys as top-level badge fields
	// BadgeDTO does not have jpg_content or png_content fields
	var raw map[string]interface{}
	json.Unmarshal(body, &raw)
	data := raw["data"].(map[string]interface{})
	badgeList := data["badges"].([]interface{})
	if len(badgeList) > 0 {
		badgeMap := badgeList[0].(map[string]interface{})
		if _, ok := badgeMap["jpg_content"]; ok {
			t.Error("backup should not include jpg_content")
		}
		if _, ok := badgeMap["png_content"]; ok {
			t.Error("backup should not include png_content")
		}
	}
}

func TestBackupWrongMethod(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	req := httptest.NewRequest(http.MethodPost, "/api/backup", nil)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Backup(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Result().StatusCode)
	}
}

func TestBackupForbiddenForNonAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	req := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	req = req.WithContext(nonAdminContext())
	rec := httptest.NewRecorder()

	h.Backup(rec, req)

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Result().StatusCode)
	}
}

// createMultipartRequest builds a multipart POST request with the given JSON as the backup_file field.
func createMultipartRequest(t *testing.T, jsonData []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("backup_file", "backup.json")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part.Write(jsonData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/restore", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// buildBackupJSON creates a valid backup document from a test DB.
func buildBackupJSON(t *testing.T, db *database.DB) []byte {
	t.Helper()

	roles, _ := db.ListRoles()
	users, _ := db.ListUsers()
	apiKeys, _ := db.ListAPIKeys()
	badges, _ := db.ListBadges()

	doc := BackupDocument{
		Metadata: BackupMetadata{Version: 1, CreatedAt: time.Now().UTC().Format(timeFormat), CreatedBy: "test", Application: "CertifyHub"},
		Data: BackupData{
			Roles:   rolesToDTOs(roles),
			Users:   usersToDTOs(users),
			APIKeys: apiKeysToDTOs(apiKeys),
			Badges:  badgesToDTOs(badges),
		},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("failed to marshal backup: %v", err)
	}
	return data
}

func TestRestoreSuccess(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	// Build a backup from current state
	backupJSON := buildBackupJSON(t, db)

	// Add an extra badge that should be removed by restore
	db.CreateBadge(&database.Badge{
		CommitID:        "extra_badge",
		Type:            "badge",
		Status:          "valid",
		Issuer:          "Test",
		IssueDate:       "2024-01-01",
		SoftwareName:    "Extra",
		SoftwareVersion: "v1.0.0",
	})

	// Verify extra badge exists
	extra, _ := db.GetBadge("extra_badge")
	if extra == nil {
		t.Fatal("extra badge should exist before restore")
	}

	// Perform restore
	req := createMultipartRequest(t, backupJSON)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "restored" {
		t.Errorf("expected status 'restored', got %v", result["status"])
	}

	// Extra badge should be gone
	extra, _ = db.GetBadge("extra_badge")
	if extra != nil {
		t.Error("extra badge should not exist after restore")
	}
}

func TestRestoreClearsCache(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	// Put something in cache
	c.Set("test-key", []byte("test-value"), time.Hour)
	if _, ok := c.Get("test-key"); !ok {
		t.Fatal("cache should have test-key before restore")
	}

	backupJSON := buildBackupJSON(t, db)
	req := createMultipartRequest(t, backupJSON)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Fatalf("expected 200, got %d: %s", rec.Result().StatusCode, string(body))
	}

	// Cache should be cleared
	if _, ok := c.Get("test-key"); ok {
		t.Error("cache should be cleared after restore")
	}
}

func TestRestoreInvalidJSON(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	req := createMultipartRequest(t, []byte("not-json"))
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Result().StatusCode)
	}
}

func TestRestoreEmptyRoles(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	doc := BackupDocument{
		Metadata: BackupMetadata{Version: 1, CreatedAt: time.Now().Format(timeFormat), CreatedBy: "test", Application: "CertifyHub"},
		Data: BackupData{
			Roles:   []RoleDTO{},
			Users:   []UserDTO{{UserID: "u1", Username: "u", Email: "u@e.com", PasswordHash: "h", FirstName: "F", LastName: "L", RoleID: "r1", CreatedAt: time.Now().Format(timeFormat), UpdatedAt: time.Now().Format(timeFormat), Status: "active"}},
			APIKeys: []APIKeyDTO{},
			Badges:  []BadgeDTO{},
		},
	}
	data, _ := json.Marshal(doc)

	req := createMultipartRequest(t, data)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty roles, got %d", rec.Result().StatusCode)
	}
}

func TestRestoreEmptyUsers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	doc := BackupDocument{
		Metadata: BackupMetadata{Version: 1, CreatedAt: time.Now().Format(timeFormat), CreatedBy: "test", Application: "CertifyHub"},
		Data: BackupData{
			Roles:   []RoleDTO{{RoleID: "r1", Name: "admin", Description: "d", Permissions: "{}", CreatedAt: time.Now().Format(timeFormat), UpdatedAt: time.Now().Format(timeFormat)}},
			Users:   []UserDTO{},
			APIKeys: []APIKeyDTO{},
			Badges:  []BadgeDTO{},
		},
	}
	data, _ := json.Marshal(doc)

	req := createMultipartRequest(t, data)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty users, got %d", rec.Result().StatusCode)
	}
}

func TestRestoreWrongMethod(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	req := httptest.NewRequest(http.MethodGet, "/api/restore", nil)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Result().StatusCode)
	}
}

func TestRestoreForbiddenForNonAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	h := NewHandler(db, logger, cache.New())

	req := httptest.NewRequest(http.MethodPost, "/api/restore", nil)
	req = req.WithContext(nonAdminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Result().StatusCode)
	}
}

func TestRestoreRollbackOnConstraintViolation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	// Get current badge count for comparison after failed restore
	badgesBefore, _ := db.ListBadges()

	// Build a backup with a user referencing a non-existent role
	doc := BackupDocument{
		Metadata: BackupMetadata{Version: 1, CreatedAt: time.Now().Format(timeFormat), CreatedBy: "test", Application: "CertifyHub"},
		Data: BackupData{
			Roles: []RoleDTO{{
				RoleID: "role1", Name: "admin", Description: "d", Permissions: "{}",
				CreatedAt: time.Now().Format(timeFormat), UpdatedAt: time.Now().Format(timeFormat),
			}},
			Users: []UserDTO{
				{UserID: "u1", Username: "user1", Email: "a@b.com", PasswordHash: "h", FirstName: "F", LastName: "L", RoleID: "role1", CreatedAt: time.Now().Format(timeFormat), UpdatedAt: time.Now().Format(timeFormat), Status: "active"},
				// Duplicate username — should cause UNIQUE constraint failure
				{UserID: "u2", Username: "user1", Email: "c@d.com", PasswordHash: "h", FirstName: "F", LastName: "L", RoleID: "role1", CreatedAt: time.Now().Format(timeFormat), UpdatedAt: time.Now().Format(timeFormat), Status: "active"},
			},
			APIKeys: []APIKeyDTO{},
			Badges:  []BadgeDTO{},
		},
	}
	data, _ := json.Marshal(doc)

	req := createMultipartRequest(t, data)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode == http.StatusOK {
		t.Error("expected failure due to constraint violation")
	}

	// Original data should be preserved (transaction rolled back)
	badgesAfter, _ := db.ListBadges()
	if len(badgesAfter) != len(badgesBefore) {
		t.Errorf("expected %d badges after rollback, got %d", len(badgesBefore), len(badgesAfter))
	}
}

func TestRestoreWithNullableFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	now := time.Now().Format(timeFormat)
	url := "https://example.com"
	note := "a note"

	doc := BackupDocument{
		Metadata: BackupMetadata{Version: 1, CreatedAt: now, CreatedBy: "test", Application: "CertifyHub"},
		Data: BackupData{
			Roles: []RoleDTO{{RoleID: "r1", Name: "admin", Description: "d", Permissions: "{}", CreatedAt: now, UpdatedAt: now}},
			Users: []UserDTO{{UserID: "u1", Username: "admin", Email: "a@b.com", PasswordHash: "h", FirstName: "F", LastName: "L", RoleID: "r1", CreatedAt: now, UpdatedAt: now, Status: "active"}},
			Badges: []BadgeDTO{
				{
					CommitID: "test_null", Type: "badge", Status: "valid", Issuer: "I", IssueDate: "2024-01-01",
					SoftwareName: "S", SoftwareVersion: "v1",
					SoftwareURL: &url, Notes: nil, PublicNote: &note,
				},
			},
		},
	}
	data, _ := json.Marshal(doc)

	req := createMultipartRequest(t, data)
	req = req.WithContext(adminContext())
	rec := httptest.NewRecorder()

	h.Restore(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rec.Result().Body)
		t.Fatalf("expected 200, got %d: %s", rec.Result().StatusCode, string(body))
	}

	badge, _ := db.GetBadge("test_null")
	if badge == nil {
		t.Fatal("badge should exist after restore")
	}
	if !badge.SoftwareURL.Valid || badge.SoftwareURL.String != url {
		t.Errorf("expected SoftwareURL %q, got %v", url, badge.SoftwareURL)
	}
	if badge.Notes.Valid {
		t.Error("expected Notes to be NULL")
	}
	if !badge.PublicNote.Valid || badge.PublicNote.String != note {
		t.Errorf("expected PublicNote %q, got %v", note, badge.PublicNote)
	}
	// Binary fields should be NULL after restore
	if badge.JPGContent != nil {
		t.Error("expected JPGContent to be nil after restore")
	}
	if badge.PNGContent != nil {
		t.Error("expected PNGContent to be nil after restore")
	}
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger, _ := zap.NewDevelopment()
	c := cache.New()
	h := NewHandler(db, logger, c)

	// Add a badge with nullable fields
	db.CreateBadge(&database.Badge{
		CommitID: "roundtrip", Type: "badge", Status: "valid", Issuer: "Test",
		IssueDate: "2024-06-01", SoftwareName: "RT", SoftwareVersion: "v2",
		SoftwareURL: sql.NullString{String: "https://rt.example.com", Valid: true},
		Notes:       sql.NullString{},
	})

	// Backup
	backupReq := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	backupReq = backupReq.WithContext(adminContext())
	backupRec := httptest.NewRecorder()
	h.Backup(backupRec, backupReq)

	if backupRec.Result().StatusCode != http.StatusOK {
		t.Fatalf("backup failed: %d", backupRec.Result().StatusCode)
	}

	backupBody, _ := io.ReadAll(backupRec.Result().Body)

	// Delete the badge we just created
	db.DeleteBadge("roundtrip")
	deleted, _ := db.GetBadge("roundtrip")
	if deleted != nil {
		t.Fatal("badge should be deleted before restore")
	}

	// Restore
	restoreReq := createMultipartRequest(t, backupBody)
	restoreReq = restoreReq.WithContext(adminContext())
	restoreRec := httptest.NewRecorder()
	h.Restore(restoreRec, restoreReq)

	if restoreRec.Result().StatusCode != http.StatusOK {
		body, _ := io.ReadAll(restoreRec.Result().Body)
		t.Fatalf("restore failed: %d: %s", restoreRec.Result().StatusCode, string(body))
	}

	// Verify roundtrip badge is back
	restored, _ := db.GetBadge("roundtrip")
	if restored == nil {
		t.Fatal("roundtrip badge should exist after restore")
	}
	if restored.SoftwareName != "RT" {
		t.Errorf("expected SoftwareName RT, got %s", restored.SoftwareName)
	}
	if !restored.SoftwareURL.Valid || restored.SoftwareURL.String != "https://rt.example.com" {
		t.Errorf("expected SoftwareURL https://rt.example.com, got %v", restored.SoftwareURL)
	}
	if restored.Notes.Valid {
		t.Error("expected Notes to be NULL after roundtrip")
	}
}
