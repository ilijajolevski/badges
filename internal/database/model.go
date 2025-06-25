package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Badge represents a badge entity in the database
// The visual difference between "badge" (small) and "certificate" (large) is determined
// by the endpoint or a rendering parameter, not by the entity itself
type Badge struct {
	CommitID        string
	Type            string // Deprecated: Kept for backward compatibility only
	Status          string // "valid", "expired", "revoked"
	Issuer          string
	IssueDate       string
	SoftwareName    string
	SoftwareVersion string
	SoftwareURL     sql.NullString
	Notes           sql.NullString
	SVGContent      sql.NullString
	ExpiryDate      sql.NullString
	IssuerURL       sql.NullString
	CustomConfig    sql.NullString
	LastReview      sql.NullString
	JPGContent      []byte
	PNGContent      []byte
	CoveredVersion  sql.NullString // semantic versioning X.X.X or git tag
	RepositoryLink  sql.NullString // code repository URL
	PublicNote      sql.NullString // long text note for public display
	InternalNote    sql.NullString // long text note for internal use only
	ContactDetails  sql.NullString // contact information for public display
	CertificateName sql.NullString // name of the certificate, e.g., "Self-Assessed Dependencies"
	SpecialtyDomain sql.NullString // specialty domain of the certificate, e.g., "SOFTWARE LICENCING"
	SoftwareSCID    sql.NullString // Software Catalogue Project ID
	SoftwareSCURL   sql.NullString // Software Catalogue Link, constructed as "https://sc.geant.org/ui/project/<software_sc_id>"
	// The following fields are for storing pre-generated outlook-specific content
	BadgeSVGContent      sql.NullString // Pre-generated SVG for badge outlook
	CertificateSVGContent sql.NullString // Pre-generated SVG for certificate outlook
	BadgePNGContent      []byte // Pre-generated PNG for badge outlook
	CertificatePNGContent []byte // Pre-generated PNG for certificate outlook
	BadgeJPGContent      []byte // Pre-generated JPG for badge outlook
	CertificateJPGContent []byte // Pre-generated JPG for certificate outlook
}

// CustomConfig represents the custom configuration for a badge
type CustomConfig struct {
	ColorLeft     string `json:"color_left,omitempty"`
	ColorRight    string `json:"color_right,omitempty"`
	TextColor     string `json:"text_color,omitempty"`
	TextColorLeft string `json:"text_color_left,omitempty"`
	TextColorRight string `json:"text_color_right,omitempty"`
	LogoURL       string `json:"logo,omitempty"`
	FontSize      int    `json:"font_size,omitempty"`
	Style         string `json:"style,omitempty"`

	// New color parameters for big certificate template
	LogoColor          string `json:"logo_color,omitempty"`
	BackgroundColor    string `json:"background_color,omitempty"`
	HorizontalBarsColor string `json:"horizontal_bars_color,omitempty"`
	TopLabelColor      string `json:"top_label_color,omitempty"`
	GradientStartColor string `json:"gradient_start_color,omitempty"`
	GradientEndColor   string `json:"gradient_end_color,omitempty"`
	BorderColor        string `json:"border_color,omitempty"`
	CertNameColor      string `json:"cert_name_color,omitempty"`
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
