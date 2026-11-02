package learning

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// validationError contains a fixed diagnostic code, never the rejected value.
type validationError struct{ code string }

func (e *validationError) Error() string { return "learning validation failed" }
func (e *validationError) Unwrap() error { return ErrInput }
func invalid(code string) error          { return &validationError{code: code} }

// DiagnosticCode is safe for logs: callers cannot supply the private error code.
func DiagnosticCode(err error) string {
	var e *validationError
	if errors.As(err, &e) {
		return e.code
	}
	return ""
}

var privateEmail = regexp.MustCompile(`(?i)[\p{L}\p{N}._%+\-]+@[\p{L}\p{N}.\-]+\.[\p{L}]{2,}`)
var privateIBAN = regexp.MustCompile(`\b[A-Z]{2}[0-9]{2}(?:[ \t]*[A-Z0-9]){11,30}`)
var phoneCandidate = regexp.MustCompile(`\+?[0-9][0-9\s()./\-]{4,}[0-9]`)
var labelledPhone = regexp.MustCompile(`(?i)\b(?:tel(?:efon)?\.?|mobil|phone|fax)\s*:?\s*\+?[0-9][0-9\s()./\-]{3,}[0-9]`)
var streetAddress = regexp.MustCompile(`(?i)\b[\p{L}][\p{L} .'’\-]{0,70}(?:stra(?:ß|ss)e|str\.|gasse|weg|allee|platz|ring|ufer|steig|damm|chaussee|promenade|gürtel|pfad|street|road|avenue|lane|drive|boulevard)\s*[0-9]{1,5}[a-z]?\b`)
var namedAddress = regexp.MustCompile(`(?i)\b(?:am|im|auf dem|an der|unter den)\s+[\p{L}][\p{L} .'’\-]{1,60}\s+[0-9]{1,5}[a-z]?\b`)
var internationalAddress = regexp.MustCompile(`(?i)\b[0-9]{1,5}[a-z]?\s+(?:[\p{L}][\p{L}'’\-]*\s+){0,5}(?:street|st\.|road|rd\.|avenue|ave\.|lane|drive|boulevard|rue|via|calle)\b`)

// A bare country/city/postcode is not a concrete private street address.
var labelledPostalAddress = regexp.MustCompile(`(?i)\b(?:adresse|anschrift|wohnadresse|privatadresse|privatanschrift|postanschrift)\s*[:#]\s*[0-9]{4,5}\s+[\p{L}][\p{L}'’\-]+`)
var personalLabel = regexp.MustCompile(`(?i)\b(?:vorname|nachname|kontoinhaber|geburtsdatum|kunden(?:nummer|nr)|bestell(?:nummer|nr)|auftrags(?:nummer|nr)|trackingnummer)\s*[:#]`)

// validatePrivacy inspects a bounded prepared candidate. Canonicalization is only
// for detection; no field is cleaned, rewritten or silently accepted after removal.
// This deliberately conservative structural guard is not a general-purpose NER.
func validatePrivacy(c Content) error {
	for _, f := range []struct {
		s   string
		max int
	}{
		{c.Topic, 150}, {c.Retrieval, 900}, {c.Question, 6000}, {c.Answer, 6000},
		{c.Rule, 6000}, {c.Evidence, 6000}, {c.Source, 6000}, {c.Limits, 6000},
	} {
		if !text(f.s, f.max) {
			return invalid("privacy_invalid_text")
		}
		s := norm.NFKC.String(html.UnescapeString(html.UnescapeString(f.s)))
		for _, r := range s {
			if unicode.Is(unicode.Cf, r) || (unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t') {
				return invalid("privacy_ambiguous_text")
			}
		}
		if privateEmail.MatchString(s) {
			return invalid("privacy_email")
		}
		if privateIBAN.MatchString(strings.ToUpper(s)) {
			return invalid("privacy_iban")
		}
		for _, number := range phoneCandidate.FindAllString(s, -1) {
			digits := 0
			for _, r := range number {
				if r >= '0' && r <= '9' {
					digits++
				}
			}
			if digits >= 7 {
				return invalid("privacy_phone")
			}
		}
		if labelledPhone.MatchString(s) {
			return invalid("privacy_phone")
		}
		if streetAddress.MatchString(s) || namedAddress.MatchString(s) || internationalAddress.MatchString(s) || labelledPostalAddress.MatchString(s) {
			return invalid("privacy_address")
		}
		if personalLabel.MatchString(s) {
			return invalid("privacy_identifier")
		}
		// Preserve the existing URL restriction and additional legacy PII patterns.
		if sensitive.MatchString(s) {
			return invalid("privacy_sensitive_text")
		}
	}
	return nil
}
