package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Badge represents a badge or certificate in the database
type Badge struct {
	CommitID        string
	Type            string // "badge" or "certificate"
	Status          string // "valid", "expired", "revoked"
	Issuer          string
	IssueDate       string
	SoftwareName    string
	SoftwareVersion string
	Notes           sql.NullString
	SVGContent      sql.NullString
	ExpiryDate      sql.NullString
	IssuerURL       sql.NullString
	CustomConfig    sql.NullString
	JPGContent      []byte
	PNGContent      []byte
}

// CustomConfig represents the custom configuration for a badge
type CustomConfig struct {
	ColorLeft   string `json:"color_left,omitempty"`
	ColorRight  string `json:"color_right,omitempty"`
	TextColor   string `json:"text_color,omitempty"`
	LogoURL     string `json:"logo,omitempty"`
	FontSize    int    `json:"font_size,omitempty"`
	Style       string `json:"style,omitempty"`
}

// GetCustomConfig parses the custom configuration JSON
func (b *Badge) GetCustomConfig() (*CustomConfig, error) {
	if !b.CustomConfig.Valid || b.CustomConfig.String == "" {
		return &CustomConfig{}, nil
	}

	var config CustomConfig
	err := json.Unmarshal([]byte(b.CustomConfig.String), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SetCustomConfig sets the custom configuration JSON
func (b *Badge) SetCustomConfig(config *CustomConfig) error {
	if config == nil {
		b.CustomConfig = sql.NullString{Valid: false}
		return nil
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	b.CustomConfig = sql.NullString{String: string(data), Valid: true}
	return nil
}

// IsValid checks if the badge is valid
func (b *Badge) IsValid() bool {
	return b.Status == "valid"
}

// IsExpired checks if the badge is expired
func (b *Badge) IsExpired() bool {
	if !b.ExpiryDate.Valid {
		return false
	}

	expiry, err := time.Parse("2006-01-02", b.ExpiryDate.String)
	if err != nil {
		return false
	}

	return time.Now().After(expiry)
}