package badge

import (
	"database/sql"
	"testing"

	"github.com/finki/badges/internal/database"
)

func TestGenerateSVG(t *testing.T) {
	// Test case 1: Badge with SoftwareVersion only (fallback case)
	t.Run("With SoftwareVersion only", func(t *testing.T) {
		// Create a test badge with only SoftwareVersion
		badge := &database.Badge{
			CommitID:        "abc123",
			Type:            "badge",
			Status:          "valid",
			Issuer:          "Test Issuer",
			IssueDate:       "2023-01-01",
			SoftwareName:    "TestApp",
			SoftwareVersion: "v1.0.0",
			// CertificateName is not set, so it should fall back to SoftwareVersion
		}

		// Create a generator
		generator := NewGenerator()

		// Generate SVG
		svg, err := generator.GenerateSVG(badge)
		if err != nil {
			t.Fatalf("Failed to generate SVG: %v", err)
		}

		// Check that the SVG was generated
		if len(svg) == 0 {
			t.Error("Generated SVG is empty")
		}

		// Check that the SVG contains the expected content
		svgStr := string(svg)
		expectedContents := []string{
			"<svg", "</svg>",
			"v1.0.0", // Should display SoftwareVersion
		}

		for _, expected := range expectedContents {
			if !contains(svgStr, expected) {
				t.Errorf("Generated SVG does not contain expected content: %s", expected)
			}
		}
	})

	// Test case 2: Badge with CertificateName set
	t.Run("With CertificateName set", func(t *testing.T) {
		// Create a test badge with CertificateName set
		badge := &database.Badge{
			CommitID:        "def456",
			Type:            "badge",
			Status:          "valid",
			Issuer:          "Test Issuer",
			IssueDate:       "2023-01-01",
			SoftwareName:    "TestApp",
			SoftwareVersion: "v1.0.0",
			CertificateName: sql.NullString{String: "Self-Assessed Dependencies", Valid: true},
		}

		// Create a generator
		generator := NewGenerator()

		// Generate SVG
		svg, err := generator.GenerateSVG(badge)
		if err != nil {
			t.Fatalf("Failed to generate SVG: %v", err)
		}

		// Check that the SVG was generated
		if len(svg) == 0 {
			t.Error("Generated SVG is empty")
		}

		// Check that the SVG contains the expected content
		svgStr := string(svg)
		expectedContents := []string{
			"<svg", "</svg>",
			"Self-Assessed Dependencies", // Should display CertificateName
		}

		for _, expected := range expectedContents {
			if !contains(svgStr, expected) {
				t.Errorf("Generated SVG does not contain expected content: %s", expected)
			}
		}
	})
}

func TestCalculateWidth(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		value     string
		fontSize  int
		wantWidth int
	}{
		{
			name:      "Empty label and value",
			label:     "",
			value:     "",
			fontSize:  12,
			wantWidth: 80, // Minimum width
		},
		{
			name:      "Label only",
			label:     "TestApp",
			value:     "",
			fontSize:  12,
			wantWidth: 53, // 7 chars * 0.6 * 12 + 10
		},
		{
			name:      "Value only",
			label:     "",
			value:     "v1.0.0",
			fontSize:  12,
			wantWidth: 46, // 6 chars * 0.6 * 12 + 10
		},
		{
			name:      "Label and value",
			label:     "TestApp",
			value:     "v1.0.0",
			fontSize:  12,
			wantWidth: 99, // (7 + 6) chars * 0.6 * 12 + 20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWidth := calculateWidth(tt.label, tt.value, tt.fontSize)

			// Allow for small rounding differences
			if abs(gotWidth-tt.wantWidth) > 1 {
				t.Errorf("calculateWidth() = %v, want %v", gotWidth, tt.wantWidth)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to get absolute value of an int
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
