package certificate

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

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
	defaultWidth       int
	defaultHeight      int

	// Default values for big certificate template
	defaultLogoColor           string
	defaultBackgroundColor     string
	defaultHorizontalBarsColor string
	defaultTopLabelColor       string
	defaultGradientStartColor  string
	defaultGradientEndColor    string
	defaultBorderColor         string
	defaultCertNameColor       string

	// Template file path
	templatePath string
}

// NewGenerator creates a new certificate generator
func NewGenerator() *Generator {
	return &Generator{
		// Default values for old template
		defaultColorBorder: "#ed1556", // GÉANT Red
		defaultColorBg:     "#003f5f", // GÉANT Blue
		defaultTextColor:   "#FFFFFF", // White text
		defaultFontSize:    18,
		defaultStyle:       "3d",
		defaultWidth:       170,
		defaultHeight:      200,

		// Default values for big certificate template
		defaultLogoColor:           "#ffffff", // White
		defaultBackgroundColor:     "#0e3f5f", // Dark blue
		defaultHorizontalBarsColor: "#e78a2d", // Orange
		defaultTopLabelColor:       "#e78a2d", // Orange
		defaultGradientStartColor:  "#ff1463", // Pink
		defaultGradientEndColor:    "#013a40", // Dark teal
		defaultBorderColor:         "#e78a2d", // Orange
		defaultCertNameColor:       "#ffffff", // White

		// Template file path
		templatePath: "templates/svg/big-template.svg",
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
	// For backward compatibility
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

	// Apply new color parameters for big certificate template
	logoColor := g.defaultLogoColor
	if config.LogoColor != "" {
		logoColor = config.LogoColor
	}

	backgroundColor := g.defaultBackgroundColor
	if config.BackgroundColor != "" {
		backgroundColor = config.BackgroundColor
	}

	horizontalBarsColor := g.defaultHorizontalBarsColor
	if config.HorizontalBarsColor != "" {
		horizontalBarsColor = config.HorizontalBarsColor
	}

	topLabelColor := g.defaultTopLabelColor
	if config.TopLabelColor != "" {
		topLabelColor = config.TopLabelColor
	}

	gradientStartColor := g.defaultGradientStartColor
	if config.GradientStartColor != "" {
		gradientStartColor = config.GradientStartColor
	}

	gradientEndColor := g.defaultGradientEndColor
	if config.GradientEndColor != "" {
		gradientEndColor = config.GradientEndColor
	}

	borderColor := g.defaultBorderColor
	if config.BorderColor != "" {
		borderColor = config.BorderColor
	}

	certNameColor := g.defaultCertNameColor
	if config.CertNameColor != "" {
		certNameColor = config.CertNameColor
	}

	// Prepare data for the template
	width := g.defaultWidth
	height := g.defaultHeight

	// Get certificate name with fallback to "Verified Dependencies"
	certificateName := "Verified Dependencies"
	if badge.CertificateName.Valid && badge.CertificateName.String != "" {
		certificateName = badge.CertificateName.String
	}

	// Split certificate name into words for multi-line display
	certNameWords := []string{}
	if certificateName != "" {
		certNameWords = splitCertificateName(certificateName)
	}

	// Get specialty domain (optional)
	var specialtyDomain string
	if badge.SpecialtyDomain.Valid {
		specialtyDomain = badge.SpecialtyDomain.String
	}

	// Read the template file
	templateContent, err := os.ReadFile(g.templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Remove XML declaration from the template content
	templateContentStr := string(templateContent)
	if len(templateContentStr) > 5 && templateContentStr[:5] == "<?xml" {
		if xmlEndIndex := bytes.Index(templateContent, []byte("?>")); xmlEndIndex != -1 {
			templateContent = templateContent[xmlEndIndex+2:]
		}
	}

	// Prepare data for the template
	data := map[string]interface{}{
		// For backward compatibility
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
		"Width":           width,
		"Height":          height,
		"CertificateName": certificateName,
		"CertNameWords":   certNameWords,
		"SpecialtyDomain": specialtyDomain,

		// New color parameters for big certificate template
		"LogoColor":           logoColor,
		"BackgroundColor":     backgroundColor,
		"HorizontalBarsColor": horizontalBarsColor,
		"TopLabelColor":       topLabelColor,
		"GradientStartColor":  gradientStartColor,
		"GradientEndColor":    gradientEndColor,
		"BorderColor":         borderColor,
		"CertNameColor":       certNameColor,
	}

	// Generate SVG using template
	tmpl := template.New("certificate").Funcs(template.FuncMap{
		"divide": func(a, b int) int {
			return a / b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"getWord": func(i int, a []string) string {
			if i < len(a) {
				return a[i]
			}
			return ""
		},
	})

	// Parse the template from the file content
	tmpl, err = tmpl.Parse(string(templateContent))
	if err != nil {
		// Fallback to the hardcoded template if the file can't be parsed
		tmpl, err = tmpl.Parse(certificateSVGTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// splitCertificateName splits a certificate name into words for multi-line display
func splitCertificateName(name string) []string {
	// For simplicity, we'll just split by space and return up to 3 words
	// In a real implementation, you might want to use a more sophisticated algorithm
	words := []string{"", "", ""}

	// Split the name by space
	parts := []string{}
	currentPart := ""
	for _, char := range name {
		if char == ' ' {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
		} else {
			currentPart += string(char)
		}
	}
	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	// Assign parts to words (up to 3)
	for i := 0; i < len(parts) && i < 3; i++ {
		words[i] = parts[i]
	}

	return words
}

// SVG template for certificates
const certificateSVGTemplate = `<svg width="{{.Width}}" height="{{.Height}}" viewBox="0 0 {{.Width}} {{.Height}}" xmlns="http://www.w3.org/2000/svg">
  <!-- Thinner GÉANT Red border -->
  <rect x="8" y="8" width="{{.Width | subtract 16}}" height="{{.Height | subtract 16}}" rx="28" fill="{{.ColorBg}}" stroke="{{.ColorBorder}}" stroke-width="8"/>

  <!-- Top label: Software name -->
  <text x="{{.Width | divide 2}}" y="70" text-anchor="middle"
        font-family="Arial, Helvetica, sans-serif"
        font-size="28"
        font-weight="bold"
        fill="{{.ColorBorder}}">
   {{.SpecialtyDomain}}
  </text>

  <!-- Badge label: Certificate Name (white on blue, no box) -->
  <text x="{{.Width | divide 2}}" y="150" text-anchor="middle"
        font-family="Arial, Helvetica, sans-serif"
        font-size="28"
        font-weight="bold"
        fill="{{.TextColor}}">
    {{.CertificateName}}
  </text>

  <!-- Specialty Domain (if provided) -->
  {{if .SpecialtyDomain}}
  <text x="{{.Width | divide 2}}" y="180" text-anchor="middle"
        font-family="Arial, Helvetica, sans-serif"
        font-size="18"
        font-weight="normal"
        fill="{{.TextColor}}">
    {{.SpecialtyDomain}}
  </text>
  {{end}}

  <!-- GÉANT logo SVG embedded -->
  <g transform="translate(90,195) scale(1.18)">
    <g>
      <g>
        <g>
          <path fill="#FFFFFF" d="M28.9,31.6c1-0.8,1.9-1.2,2.7-1.2c1.7,0.1,2.2,1.2,2.3,1.9c-0.4,0.1-7.8,2.6-8.2,2.7
            c-0.1-0.1-0.2-0.3-0.4-0.4C25.8,34.4,28.9,31.6,28.9,31.6z"/>
          <path fill="#FFFFFF" d="M1.5,47.5c0,8.4,3.7,12.7,11,12.7c4.8,0,7.7-2.1,7.8-2.2l0.2-0.2V46.2H10.5v3.2c0,0,5,0,6,0c0,0.9,0,6,0,6.5
            c-0.6,0.3-2.1,1-4.2,1c-4.3,0-6.4-3.1-6.4-9.4c0-3.6,1-8,6-8c3.3,0,4.4,1.9,4.4,3.7v0.6h4.5v-0.6c0-4.1-3.6-6.9-8.9-6.9
            C5.2,36.3,1.5,40.4,1.5,47.5z"/>
          <path fill="#FFFFFF" d="M36.4,36.7H23.2v23.1h14.1v-3.2c0,0-9,0-10,0c0-0.9,0-6.3,0-7.3c1,0,9.2,0,9.2,0v-3.2c0,0-8.2,0-9.2,0
            c0-0.9,0-5.3,0-6.2c1,0,9.7,0,9.7,0v-3.2H36.4z"/>
          <g>
            <path fill="#FFFFFF" d="M95.8,36.7h-20c0,0,0,13.9,0,17.1c-1.6-2.8-9.9-17.1-9.9-17.1h-4.7v23.1h3.9c0,0,0-13.9,0-17.1
              c1.6,2.8,9.9,17.1,9.9,17.1h4.7c0,0,0-18.8,0-19.9c0.9,0,5.5,0,6.4,0c0,1.1,0,19.9,0,19.9h4.1c0,0,0-18.8,0-19.9
              c0.9,0,6.2,0,6.2,0v-3.2H95.8z"/>
          </g>
          <path fill="#FFFFFF" d="M51.5,36.7h-0.4h-4l-8.7,23.1h4.2c0,0,2.3-6.1,2.5-6.8c0.7,0,7.8,0,8.5,0c0.3,0.7,2.6,6.8,2.6,6.8h4.2
            L51.5,36.7z M46.3,49.8c0.4-1.1,2.3-6.7,3-8.7c0.7,2,2.6,7.5,3,8.7C51.2,49.8,47.4,49.8,46.3,49.8z"/>
        </g>
        <g>
          <path fill="#FFFFFF" d="M134.7,14.7c-15.2-18.8-76.9,7.8-93.4,14.8c-1.2,0.5-2.7,0.4-3.6-1.3c0.7,1.7,2,2.3,3.7,1.7
            c22-8.8,75.2-27.9,88.5-10.5c6,7.9,4.3,17.6-2.3,31.3c-0.3,0.6-0.5,1-0.6,1.1c0,0,0,0.1-0.1,0.1c0,0,0,0.1-0.1,0.1
            c-0.5,0.8-1.2,1.3-1.8,1.5c0.8,0,1.6-0.4,2.2-1.4c0.2-0.3,0.4-0.6,0.6-1l0,0C137.7,34.7,141.1,22.5,134.7,14.7z"/>
        </g>
        <g>
          <path fill="#E5004B" d="M123.2,52.6c-0.2-0.2-3-2.6-5.7-5.2C103,33.8,59.4-8.4,40.3,2.7c-5.4,3.1-6.3,12.2-3,24.3c0,0,0,0,0,0.1v0
            c0,0.2,0.1,0.3,0.1,0.5c0.4,1.3,1.3,2.1,2.4,2.1c-0.8-0.2-1.5-0.8-1.9-1.8c-0.1-0.1-0.1-0.3-0.1-0.4c-0.1-0.2-0.1-0.4-0.2-0.7
            l0,0c0-0.1-0.1-0.3-0.1-0.4c-1.8-10.3,0.4-17,4.5-19.8c15.3-10,52,21.6,70.4,37.5c4.2,3.7,9,7.7,10.5,8.8c2.1,1.6,3.8-0.2,4.3-1
            C126.6,53,124.8,54,123.2,52.6z"/>
        </g>
      </g>
    </g>
  </g>

  <!-- GEANT slogan -->
  <text x="{{.Width | divide 2}}" y="295" text-anchor="middle"
        font-family="Arial, Helvetica, sans-serif"
        font-size="18"
        fill="{{.TextColor}}">
    Networks • Services • People
  </text>
</svg>`
