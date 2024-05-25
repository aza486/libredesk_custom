package utils

import (
	"crypto/rand"
	"fmt"
	"net/textproto"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	reSpaces = regexp.MustCompile(`[\s]+`)
)

// RandomAlNumString generates a random alphanumeric string of length n.
func RandomAlNumString(n int) (string, error) {
	const (
		dictionary = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	)

	var bytes = make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// GenerateRandomNumericString generates a random digit string of length n.
func GenerateRandomNumericString(n int) (string, error) {
	const (
		dictionary = "0123456789"
	)

	var bytes = make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// GeneratePassword generates a secure password of specified length.
func GeneratePassword(len int) ([]byte, error) {
	randomString, err := RandomAlNumString(len)

	if err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomString), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return hashedPassword, nil
}

// InArray checks if an element of type T is present in a slice of type T.
func InArray[T comparable](val T, vals []T) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}
	return false
}

// MakeFilename makes a filename from the given string.
func MakeFilename(fName string) string {
	name := strings.TrimSpace(fName)
	if name == "" {
		name, _ = RandomAlNumString(10)
	}
	// replace whitespace with "-"
	name = reSpaces.ReplaceAllString(name, "-")
	return filepath.Base(name)
}

// MakeAttachmentHeader
func MakeAttachmentHeader(filename, encoding, contentType string) textproto.MIMEHeader {
	if encoding == "" {
		encoding = "base64"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", "attachment; filename="+filename)
	h.Set("Content-Type", fmt.Sprintf("%s; name=\""+filename+"\"", contentType))
	h.Set("Content-Transfer-Encoding", encoding)
	return h
}

// SplitName splits a full name into first name and last name.
func SplitName(fullName string) (firstName string, lastName string) {
	parts := strings.Fields(fullName)
	if len(parts) > 1 {
		lastName = parts[len(parts)-1]
		firstName = strings.Join(parts[:len(parts)-1], " ")
	} else if len(parts) == 1 {
		firstName = parts[0]
	}
	return firstName, lastName
}

// BackoffDelay introduces a delay between actions with backoff behavior.
func BackoffDelay(try int, dur time.Duration) {
	if try > 0 {
		<-time.After(time.Duration(try) * time.Duration(dur))
	}
}
