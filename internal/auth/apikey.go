package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	
	"github.com/finki/badges/internal/database"
)

// APIKeyPrefix is the prefix for all API keys
const APIKeyPrefix = "bsvc_"

// APIKeyLength is the length of the API key in bytes (before encoding)
const APIKeyLength = 32 // 256 bits

// ErrInvalidAPIKey is returned when an API key is invalid
var ErrInvalidAPIKey = errors.New("invalid API key")

// GenerateAPIKey generates a new API key with sufficient entropy
func GenerateAPIKey() (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, APIKeyLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as hex
	apiKey := APIKeyPrefix + hex.EncodeToString(randomBytes)

	return apiKey, nil
}

// HashAPIKey hashes an API key using bcrypt
func HashAPIKey(apiKey string) (string, error) {
	// Validate API key format
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return "", ErrInvalidAPIKey
	}

	// Hash API key with bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyAPIKey verifies an API key against a hash
func VerifyAPIKey(hashedAPIKey, apiKey string) error {
	// Validate API key format
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return ErrInvalidAPIKey
	}

	// Compare API keys
	err := bcrypt.CompareHashAndPassword([]byte(hashedAPIKey), []byte(apiKey))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidAPIKey
		}
		return fmt.Errorf("failed to verify API key: %w", err)
	}

	return nil
}

// ValidateIPRestriction checks if a client IP is allowed by the IP restrictions
func ValidateIPRestriction(clientIP string, restrictions []string) bool {
	// If no restrictions, allow all
	if len(restrictions) == 0 {
		return true
	}

	// Clean up client IP (remove port if present)
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}

	// Check if client IP is allowed
	for _, restriction := range restrictions {
		// Check for exact match
		if restriction == clientIP {
			return true
		}

		// Check for CIDR match
		if strings.Contains(restriction, "/") {
			_, ipNet, err := net.ParseCIDR(restriction)
			if err != nil {
				continue
			}
			ip := net.ParseIP(clientIP)
			if ip != nil && ipNet.Contains(ip) {
				return true
			}
		}

		// Check for prefix match (e.g., "192.168.1.")
		if strings.HasSuffix(restriction, ".") && strings.HasPrefix(clientIP, restriction) {
			return true
		}
	}

	return false
}

// ConvertDBPermissionsToMap converts database APIKeyPermissions to a map
func ConvertDBPermissionsToMap(permissions *database.APIKeyPermissions) map[string]map[string]bool {
	if permissions == nil {
		return map[string]map[string]bool{}
	}

	return map[string]map[string]bool{
		"badges": {
			"read":  permissions.Badges.Read,
			"write": permissions.Badges.Write,
		},
	}
}

// GetAPIKeyValidator returns a function that validates API keys against the database
func GetAPIKeyValidator(db interface {
	GetAPIKeyByKey(hashedKey string) (*database.APIKey, error)
	UpdateAPIKeyLastUsed(apiKeyID string, lastUsed time.Time) error
}) func(string) (*APIKeyInfo, error) {
	return func(apiKey string) (*APIKeyInfo, error) {
		// Verify API key format
		if !strings.HasPrefix(apiKey, APIKeyPrefix) {
			return nil, ErrInvalidAPIKey
		}

		// Get API key from database by the raw key
		// Note: In a real implementation, we would hash the key first
		// but for simplicity, we're using the raw key for now
		dbAPIKey, err := db.GetAPIKeyByKey(apiKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get API key: %w", err)
		}

		// Check if API key exists
		if dbAPIKey == nil {
			return nil, nil
		}

		// Get IP restrictions
		ipRestrictions, err := dbAPIKey.GetIPRestrictions()
		if err != nil {
			return nil, fmt.Errorf("failed to get IP restrictions: %w", err)
		}

		// Get permissions
		dbPermissions, err := dbAPIKey.GetPermissions()
		if err != nil {
			return nil, fmt.Errorf("failed to get permissions: %w", err)
		}
		permissions := ConvertDBPermissionsToMap(dbPermissions)

		// Update last used timestamp
		err = db.UpdateAPIKeyLastUsed(dbAPIKey.APIKeyID, time.Now())
		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to update API key last used: %v\n", err)
		}

		// Create API key info
		apiKeyInfo := &APIKeyInfo{
			ID:             dbAPIKey.APIKeyID,
			UserID:         dbAPIKey.UserID,
			Status:         dbAPIKey.Status,
			ExpiresAt:      dbAPIKey.ExpiresAt,
			IPRestrictions: ipRestrictions,
			Permissions:    permissions,
		}

		return apiKeyInfo, nil
	}
}