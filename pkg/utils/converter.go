package utils

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"os/exec"

	"github.com/disintegration/imaging"
)

// SVGToJPG converts SVG content to JPG format
func SVGToJPG(svgContent []byte, width, height int) ([]byte, error) {
	// Convert SVG to PNG first (using rsvg-convert or similar)
	pngData, err := svgToPNG(svgContent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert SVG to PNG: %w", err)
	}

	// Decode PNG
	img, err := imaging.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Resize if needed
	if width > 0 && height > 0 {
		img = imaging.Resize(img, width, height, imaging.Lanczos)
	}

	// Encode to JPG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPG: %w", err)
	}

	return buf.Bytes(), nil
}

// SVGToPNG converts SVG content to PNG format
func SVGToPNG(svgContent []byte, width, height int) ([]byte, error) {
	// Convert SVG to PNG
	pngData, err := svgToPNG(svgContent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert SVG to PNG: %w", err)
	}

	// If no resizing needed, return as is
	if width <= 0 || height <= 0 {
		return pngData, nil
	}

	// Decode PNG for resizing
	img, err := imaging.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Resize
	img = imaging.Resize(img, width, height, imaging.Lanczos)

	// Encode back to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// svgToPNG converts SVG to PNG using external tools
// This is a helper function that tries multiple methods
func svgToPNG(svgContent []byte) ([]byte, error) {
	// Try using rsvg-convert if available (usually on Linux/macOS)
	pngData, err := convertWithRSVG(svgContent)
	if err == nil {
		return pngData, nil
	}

	// Fallback to other methods if rsvg-convert fails or is not available
	// For a production system, you might want to use a pure Go SVG renderer
	// or ensure that the required tools are installed on the server

	// For now, we'll return an error if rsvg-convert fails
	return nil, fmt.Errorf("failed to convert SVG to PNG: %w", err)
}

// convertWithRSVG uses rsvg-convert to convert SVG to PNG
func convertWithRSVG(svgContent []byte) ([]byte, error) {
	// Check if rsvg-convert is available
	_, err := exec.LookPath("rsvg-convert")
	if err != nil {
		return nil, fmt.Errorf("rsvg-convert not found: %w", err)
	}

	// Create command
	cmd := exec.Command("rsvg-convert", "-f", "png")

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start rsvg-convert: %w", err)
	}

	// Write SVG to stdin
	_, err = io.WriteString(stdin, string(svgContent))
	if err != nil {
		return nil, fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("rsvg-convert failed: %w", err)
	}

	return stdout.Bytes(), nil
}
