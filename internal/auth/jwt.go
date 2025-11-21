package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT secret key - in production, this should be loaded from environment variables or a secure configuration
var jwtSecret = []byte("your-secret-key-here")

// TokenExpiration is the duration for which a token is valid
// Adjusted to 15 minutes per requirements
const TokenExpiration = 15 * time.Minute

// Claims represents the JWT claims
type Claims struct {
	UserID      string `json:"sub"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Permissions struct {
		Badges struct {
			Read   bool `json:"read"`
			Write  bool `json:"write"`
			Delete bool `json:"delete"`
		} `json:"badges"`
		Users struct {
			Read   bool `json:"read"`
			Write  bool `json:"write"`
			Delete bool `json:"delete"`
		} `json:"users"`
		APIKeys struct {
			Read   bool `json:"read"`
			Write  bool `json:"write"`
			Delete bool `json:"delete"`
		} `json:"api_keys"`
	} `json:"permissions"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, username, email, role string, permissions map[string]interface{}) (string, time.Time, error) {
	// Set expiration time
	expirationTime := time.Now().Add(TokenExpiration)

	// Create claims
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "certificates.software.geant.org",
		},
	}

	// Set permissions
	if badgePerms, ok := permissions["badges"].(map[string]interface{}); ok {
		if read, ok := badgePerms["read"].(bool); ok {
			claims.Permissions.Badges.Read = read
		}
		if write, ok := badgePerms["write"].(bool); ok {
			claims.Permissions.Badges.Write = write
		}
		if delete, ok := badgePerms["delete"].(bool); ok {
			claims.Permissions.Badges.Delete = delete
		}
	}

	if userPerms, ok := permissions["users"].(map[string]interface{}); ok {
		if read, ok := userPerms["read"].(bool); ok {
			claims.Permissions.Users.Read = read
		}
		if write, ok := userPerms["write"].(bool); ok {
			claims.Permissions.Users.Write = write
		}
		if delete, ok := userPerms["delete"].(bool); ok {
			claims.Permissions.Users.Delete = delete
		}
	}

	if apiKeyPerms, ok := permissions["api_keys"].(map[string]interface{}); ok {
		if read, ok := apiKeyPerms["read"].(bool); ok {
			claims.Permissions.APIKeys.Read = read
		}
		if write, ok := apiKeyPerms["write"].(bool); ok {
			claims.Permissions.APIKeys.Write = write
		}
		if delete, ok := apiKeyPerms["delete"].(bool); ok {
			claims.Permissions.APIKeys.Delete = delete
		}
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// ValidateToken validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// Validate token
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// RefreshToken refreshes a JWT token
func RefreshToken(tokenString string) (string, time.Time, error) {
	// Validate token
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return "", time.Time{}, err
	}

	// Create permissions map
	permissions := map[string]interface{}{
		"badges": map[string]interface{}{
			"read":   claims.Permissions.Badges.Read,
			"write":  claims.Permissions.Badges.Write,
			"delete": claims.Permissions.Badges.Delete,
		},
		"users": map[string]interface{}{
			"read":   claims.Permissions.Users.Read,
			"write":  claims.Permissions.Users.Write,
			"delete": claims.Permissions.Users.Delete,
		},
		"api_keys": map[string]interface{}{
			"read":   claims.Permissions.APIKeys.Read,
			"write":  claims.Permissions.APIKeys.Write,
			"delete": claims.Permissions.APIKeys.Delete,
		},
	}

	// Generate new token
	return GenerateToken(claims.UserID, claims.Username, claims.Email, claims.Role, permissions)
}

// SetJWTSecret sets the JWT secret key
func SetJWTSecret(secret string) {
	jwtSecret = []byte(secret)
}