package certificate

import (
	"reflect"
	"testing"
)

func TestSplitSoftwareNameLines(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		maxPerLine int
		maxLines   int
		want       []string
	}{
		{"short name fits one line", "Shibboleth", 16, 3, []string{"Shibboleth"}},
		{"exactly at limit", "1234567890123456", 16, 3, []string{"1234567890123456"}},
		{"two words two lines", "eduGAIN Reporting", 16, 3, []string{"eduGAIN", "Reporting"}},
		{"three words three lines", "eduGAIN Reporting ecosystem", 16, 3, []string{"eduGAIN", "Reporting", "ecosystem"}},
		{"two words fit one line", "Hello World", 16, 3, []string{"Hello World"}},
		{"empty name", "", 16, 3, []string{""}},
		{"single long word", "VeryLongSoftwareNameHere", 16, 3, []string{"VeryLongSoftwareNameHere"}},
		{"four words limited to 3 lines", "A B C D", 2, 3, []string{"A", "B", "C D"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitSoftwareNameLines(tt.input, tt.maxPerLine, tt.maxLines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitSoftwareNameLines(%q, %d, %d) = %v, want %v", tt.input, tt.maxPerLine, tt.maxLines, got, tt.want)
			}
		})
	}
}

func TestCalcSoftwareNameFontSize(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		threeLines bool
		want       int
	}{
		{"short 1-line", []string{"Shibboleth"}, false, 16},
		{"14 chars 1-line", []string{"12345678901234"}, false, 16},
		{"15 chars 1-line scales down", []string{"SoftwareCertHub"}, false, 14},
		{"20 chars 1-line", []string{"12345678901234567890"}, false, 11},
		{"3-line within limit", []string{"eduGAIN", "Reporting", "ecosystem"}, true, 14},
		{"3-line long word", []string{"eduGAIN", "VeryLongReporting", "ecosystem"}, true, 13},
		{"2-line long word", []string{"SoftwareCertHub", "Extended"}, false, 14},
		{"minimum font size", []string{"AVeryVeryVeryVeryLongName"}, false, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcSoftwareNameFontSize(tt.lines, tt.threeLines)
			if got != tt.want {
				t.Errorf("calcSoftwareNameFontSize(%v, %v) = %d, want %d", tt.lines, tt.threeLines, got, tt.want)
			}
		})
	}
}
