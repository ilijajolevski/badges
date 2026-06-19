package auth

import (
	"errors"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the cost factor for bcrypt password hashing
// Higher values are more secure but slower
const BcryptCost = 12

// MinPasswordLength is the minimum length for passwords
const MinPasswordLength = 8

// Common password validation errors
var (
	ErrPasswordTooShort     = errors.New("password is too short, minimum length is 8 characters")
	ErrPasswordNoUpper      = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower      = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber     = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial    = errors.New("password must contain at least one special character")
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	// Validate password before hashing
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	// Hash password with bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword compares a plaintext password with a hashed password
func VerifyPassword(hashedPassword, password string) error {
	// Compare passwords
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("incorrect password")
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}

	return nil
}

// ValidatePassword checks if a password meets the security requirements
func ValidatePassword(password string) error {
	// Check password length
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	// Check for uppercase letters
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return ErrPasswordNoUpper
	}

	// Check for lowercase letters
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return ErrPasswordNoLower
	}

	// Check for numbers
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return ErrPasswordNoNumber
	}

	// Check for special characters
	if !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		return ErrPasswordNoSpecial
	}

	return nil
}

// GenerateRandomPassword generates a random password that meets the security requirements
func GenerateRandomPassword() (string, error) {
	// Implementation of random password generation
	// This is a placeholder - in a real implementation, you would use a secure
	// random number generator to create a password that meets all requirements
	
	// For now, return a fixed password that meets all requirements
	// In production, this should be replaced with actual random generation
	return "Secure-Password-123", nil
}