package stringutils

import (
	"crypto/rand"
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

// RandomNumericString generates a random digit string of length n.
func RandomNumericString(n int) (string, error) {
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
