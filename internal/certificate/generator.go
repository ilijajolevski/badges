package certificate

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/finki/badges/internal/database"
)

// Generator is responsible for generating certificate SVGs
type Generator struct {
	// Default values for certificate customization
	defaultColorBorder string
	defaultColorBg     string
	defaultTextColor   string
	defaultFontSize    int
	defaultStyle       string
}

// NewGenerator creates a new certificate generator
func NewGenerator() *Generator {
	return &Generator{
		defaultColorBorder: "#4B6CB7",
		defaultColorBg:     "#FFFFFF",
		defaultTextColor:   "#333333",
		defaultFontSize:    14,
		defaultStyle:       "3d",
	}
}

// GenerateSVG generates an SVG certificate
func (g *Generator) GenerateSVG(badge *database.Badge) ([]byte, error) {
	// Get custom configuration
	config, err := badge.GetCustomConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get custom config: %w", err)
	}

	// Apply default values if not specified
	colorBorder := g.defaultColorBorder
	if config.ColorLeft != "" {
		colorBorder = config.ColorLeft
	}

	colorBg := g.defaultColorBg
	if config.ColorRight != "" {
		colorBg = config.ColorRight
	}

	textColor := g.defaultTextColor
	if config.TextColor != "" {
		textColor = config.TextColor
	}

	fontSize := g.defaultFontSize
	if config.FontSize > 0 {
		fontSize = config.FontSize
	}

	style := g.defaultStyle
	if config.Style != "" {
		style = config.Style
	}

	// Prepare data for the template
	data := map[string]interface{}{
		"ColorBorder":     colorBorder,
		"ColorBg":         colorBg,
		"TextColor":       textColor,
		"FontSize":        fontSize,
		"Style":           style,
		"LogoURL":         config.LogoURL,
		"SoftwareName":    badge.SoftwareName,
		"SoftwareVersion": badge.SoftwareVersion,
		"Issuer":          badge.Issuer,
		"IssueDate":       badge.IssueDate,
		"CommitID":        badge.CommitID,
		"HasShadow":       style == "3d",
	}

	// Generate SVG using template
	tmpl, err := template.New("certificate").Parse(certificateSVGTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// SVG template for certificates
const certificateSVGTemplate = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="400" height="300">
  <rect width="400" height="300" rx="10" fill="{{.ColorBg}}" stroke="{{.ColorBorder}}" stroke-width="5"/>
  
  {{if .HasShadow}}
  <filter id="shadow">
    <feDropShadow dx="0" dy="2" stdDeviation="1" flood-color="#000" flood-opacity="0.3"/>
  </filter>
  <rect width="400" height="300" rx="10" fill="transparent" filter="url(#shadow)"/>
  {{end}}
  
  <g fill="{{.TextColor}}" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif">
    {{if .LogoURL}}
    <image x="175" y="30" width="50" height="50" xlink:href="{{.LogoURL}}"/>
    {{end}}
    
    <text x="200" y="100" font-size="24" font-weight="bold">Certificate</text>
    
    <text x="200" y="140" font-size="{{.FontSize}}">{{.SoftwareName}} {{.SoftwareVersion}}</text>
    
    <text x="200" y="180" font-size="{{.FontSize}}">Issued by: {{.Issuer}}</text>
    
    <text x="200" y="210" font-size="{{.FontSize}}">Date: {{.IssueDate}}</text>
    
    <text x="200" y="250" font-size="10">ID: {{.CommitID}}</text>
  </g>
</svg>`