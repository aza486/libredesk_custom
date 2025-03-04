// Package stringutil provides string utility functions.
package stringutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/k3a/html2text"
)

const (
	PasswordDummy = "•"
)

var (
	regexpNonAlNum = regexp.MustCompile(`[^a-zA-Z0-9\-_\.]+`)
	regexpSpaces   = regexp.MustCompile(`[\s]+`)
)

// HTML2Text converts HTML to text.
func HTML2Text(html string) string {
	return strings.TrimSpace(html2text.HTML2Text(html))
}

// SanitizeFilename sanitizes the provided filename.
func SanitizeFilename(fName string) string {
	// Trim whitespace.
	name := strings.TrimSpace(fName)

	// Replace whitespace and "/" with "-"
	name = regexpSpaces.ReplaceAllString(name, "-")

	// Remove or replace any non-alphanumeric characters
	name = regexpNonAlNum.ReplaceAllString(name, "")

	// Convert to lowercase
	name = strings.ToLower(name)
	return filepath.Base(name)
}

// RandomAlphanumeric generates a random alphanumeric string of length n.
func RandomAlphanumeric(n int) (string, error) {
	const dictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// RandomNumeric generates a random numeric string of length n.
func RandomNumeric(n int) (string, error) {
	const dictionary = "0123456789"

	bytes := make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// GetPathFromURL extracts the path from a URL.
func GetPathFromURL(u string) (string, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return parsedURL.Path, nil
}

// RemoveEmpty removes empty strings from a slice of strings.
func RemoveEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

// GenerateEmailMessageID generates a RFC-compliant Message-ID for an email.
func GenerateEmailMessageID(messageID string, fromAddress string) (string, error) {
	if messageID == "" {
		return "", fmt.Errorf("messageID cannot be empty")
	}

	// Parse from address
	addr, err := mail.ParseAddress(fromAddress)
	if err != nil {
		return "", fmt.Errorf("invalid from address: %w", err)
	}

	// Extract domain with validation
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid domain in from address")
	}
	domain := parts[1]

	// Generate cryptographic random component
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Sanitize messageID for email Message-ID
	cleaner := regexp.MustCompile(`[^\w.-]`) // Allow only alphanum, ., -, _
	cleanmessageID := cleaner.ReplaceAllString(messageID, "_")

	// Ensure cleaned messageID isn't empty
	if cleanmessageID == "" {
		return "", fmt.Errorf("messageID became empty after sanitization")
	}

	// Build RFC-compliant Message-ID
	return fmt.Sprintf("%s-%d-%s@%s",
		cleanmessageID,
		time.Now().UnixNano(), // Nanosecond precision
		strings.TrimRight(base64.URLEncoding.EncodeToString(random), "="), // URL-safe base64 without padding
		domain,
	), nil
}
