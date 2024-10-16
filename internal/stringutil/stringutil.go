// Package stringutil provides string utility functions.
package stringutil

import (
	"crypto/rand"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/k3a/html2text"
)

var (
	regexpNonAlNum = regexp.MustCompile(`[^a-zA-Z0-9\-_\.]+`)
	regexpSpaces   = regexp.MustCompile(`[\s]+`)
)

// SanitizeAndTruncate removes HTML tags, trims whitespace, makes the text human-readable, and shortens the content to a specified maximum length, appending "..." if truncated.
func SanitizeAndTruncate(content string, maxLen int) string {
	plain := strings.TrimSpace(html2text.HTML2Text(content))
	if len(plain) > maxLen {
		plain = plain[:maxLen] + "..."
	}
	return plain
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

// RandomAlNumString generates a random alphanumeric string of length n.
func RandomAlNumString(n int) (string, error) {
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

// RandomNumericString generates a random numeric string of length n.
func RandomNumericString(n int) (string, error) {
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

func GetPathFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return parsedURL.Path, nil
}