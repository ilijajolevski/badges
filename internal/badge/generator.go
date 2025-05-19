package badge

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/finki/badges/internal/database"
)

// Generator is responsible for generating badge SVGs
type Generator struct {
	// Default values for badge customization
	defaultColorLeft   string
	defaultColorRight  string
	defaultTextColor   string
	defaultFontSize    int
	defaultStyle       string
}

// NewGenerator creates a new badge generator
func NewGenerator() *Generator {
	return &Generator{
		defaultColorLeft:   "#333",
		defaultColorRight:  "#4CAF50",
		defaultTextColor:   "#FFFFFF",
		defaultFontSize:    12,
		defaultStyle:       "3d",
	}
}

// GenerateSVG generates an SVG badge
func (g *Generator) GenerateSVG(badge *database.Badge) ([]byte, error) {
	// Get custom configuration
	config, err := badge.GetCustomConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get custom config: %w", err)
	}

	// Apply default values if not specified
	colorLeft := g.defaultColorLeft
	if config.ColorLeft != "" {
		colorLeft = config.ColorLeft
	}

	colorRight := g.defaultColorRight
	if config.ColorRight != "" {
		colorRight = config.ColorRight
	}

	textColor := g.defaultTextColor
	if config.TextColor != "" {
		textColor = config.TextColor
	}

	textColorLeft := textColor
	if config.TextColorLeft != "" {
		textColorLeft = config.TextColorLeft
	}

	textColorRight := textColor
	if config.TextColorRight != "" {
		textColorRight = config.TextColorRight
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
		"ColorLeft":       colorLeft,
		"ColorRight":      colorRight,
		"TextColor":       textColor,
		"TextColorLeft":   textColorLeft,
		"TextColorRight":  textColorRight,
		"FontSize":        fontSize,
		"Style":           style,
		"LogoURL":         config.LogoURL,
		"Label":           badge.SoftwareName,
		"Value":           badge.SoftwareVersion,
		"Width":           calculateWidth(badge.SoftwareName, badge.SoftwareVersion, fontSize),
		"LeftWidth":       calculateWidth(badge.SoftwareName, "", fontSize),
		"RightWidth":      calculateWidth("", badge.SoftwareVersion, fontSize),
		"HasShadow":       style == "3d",
	}

	// Generate SVG using template
	tmpl := template.New("badge").Funcs(template.FuncMap{
		"div": func(a, b int) int {
			return a / b
		},
		"add": func(a, b int) int {
			return a + b
		},
	})

	tmpl, err = tmpl.Parse(badgeSVGTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// calculateWidth calculates the width of the badge based on the text length
func calculateWidth(label, value string, fontSize int) int {
	// Approximate width calculation based on character count
	// This is a simple approximation, a more accurate calculation would consider font metrics
	charWidth := float64(fontSize) * 0.6

	labelLen := float64(len(label))
	valueLen := float64(len(value))

	if label != "" && value != "" {
		// Both label and value
		return int((labelLen + valueLen) * charWidth) + 20 // Add padding
	} else if label != "" {
		// Only label
		return int(labelLen * charWidth) + 10 // Add padding
	} else if value != "" {
		// Only value
		return int(valueLen * charWidth) + 10 // Add padding
	}

	return 80 // Minimum width
}

// SVG template for badges
const badgeSVGTemplate = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="{{.Width}}" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="a">
    <rect width="{{.Width}}" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#a)">
    <rect width="{{.LeftWidth}}" height="20" fill="{{.ColorLeft}}"/>
    <rect x="{{.LeftWidth}}" width="{{.RightWidth}}" height="20" fill="{{.ColorRight}}"/>
    <rect width="{{.Width}}" height="20" fill="url(#b)"/>
  </g>
  <g text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="{{.FontSize}}">
    {{if .LogoURL}}
    <image x="5" y="3" width="14" height="14" xlink:href="{{.LogoURL}}"/>
    <text x="{{div .LeftWidth 2}}" y="15" fill="{{.TextColorLeft}}">{{.Label}}</text>
    {{else}}
    <text x="{{div .LeftWidth 2}}" y="15" fill="{{.TextColorLeft}}">{{.Label}}</text>
    {{end}}
    <text x="{{add .LeftWidth (div .RightWidth 2)}}" y="15" fill="{{.TextColorRight}}">{{.Value}}</text>
  </g>
  {{if .HasShadow}}
  <filter id="shadow">
    <feDropShadow dx="0" dy="1" stdDeviation="0.5" flood-color="#000" flood-opacity="0.3"/>
  </filter>
  <rect width="{{.Width}}" height="20" rx="3" fill="transparent" filter="url(#shadow)"/>
  {{end}}
</svg>`
