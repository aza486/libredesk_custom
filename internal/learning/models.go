// Package learning manages human review separately from technical RAG publication.
package learning

import (
	"context"
	"encoding/json"
	"errors"
	"golang.org/x/text/unicode/norm"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var ErrForbidden = errors.New("conversation access required")

var ErrInput = errors.New("invalid review input")
var ErrConflict = errors.New("review changed or action not allowed")
var ErrUnavailable = errors.New("learning integration unavailable")
var Categories = []string{"bestellstatus", "bestellung", "versandstatus", "rechnung", "produktfragen", "firmenfragen", "reklamation", "spam", "human_needed"}
var Types = []string{"employee_fact", "factual_review", "response_example", "classification_example", "style_rule"}

type Content struct {
	Type                string   `json:"source_type"`
	Category            string   `json:"category"`
	Topic               string   `json:"topic"`
	Retrieval           string   `json:"retrieval_text"`
	Question            string   `json:"customer_question"`
	Answer              string   `json:"human_answer"`
	Rule                string   `json:"rule"`
	Evidence            string   `json:"evidence"`
	Source              string   `json:"source"`
	Limits              string   `json:"limits"`
	OriginalCategories  []string `json:"original_categories"`
	CorrectedCategories []string `json:"corrected_categories"`
}
type Checks struct {
	Privacy    bool `json:"privacy"`
	Content    bool `json:"content"`
	General    bool `json:"general"`
	Correction bool `json:"correction"`
	StyleOnly  bool `json:"style_only"`
}
type Mutation struct {
	decisionV2 bool
	Action     string `json:"action"`
	Revision   int64  `json:"revision"`
	Reason     string `json:"reason"`
	Checks     Checks `json:"checks"`
}
type DisplayContext struct {
	Question string `json:"customer_question"`
	Answer   string `json:"human_answer"`
}

type Item struct {
	SentAnswer     string            `json:"sent_answer,omitempty"`
	Context        *DisplayContext   `json:"context,omitempty"`
	ReviewReady    bool              `json:"review_ready"`
	RecordType     string            `json:"record_type"`
	LearningStatus string            `json:"learning_status,omitempty"`
	CandidateCount int               `json:"candidate_count"`
	ReviewExists   bool              `json:"review_exists"`
	AISuggestion   string            `json:"ai_suggestion,omitempty"`
	ID             string            `json:"id"`
	Content        Content           `json:"content"`
	Quality        string            `json:"quality_status"`
	Approved       bool              `json:"approved"`
	Active         bool              `json:"active"`
	Publish        string            `json:"publish_state"`
	Decision       string            `json:"decision"`
	Revision       int64             `json:"revision"`
	Reason         string            `json:"feedback_reason"`
	Rationale      string            `json:"rationale"`
	Provenance     json.RawMessage   `json:"provenance"`
	Created        string            `json:"created_at"`
	BlockReason    string            `json:"block_reason"`
	History        []json.RawMessage `json:"history"`
	Original       json.RawMessage   `json:"original"`
}
type Filter struct {
	Tab, Type, Status string
	Page              int
}
type Page struct {
	Items   []Item `json:"items"`
	Page    int    `json:"page"`
	HasMore bool   `json:"has_more"`
}
type Embedder func(context.Context, string) ([]float64, error)
type Anonymizer func(context.Context, string) (string, error)

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func text(s string, max int) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, 0) && utf8.RuneCountInString(s) <= max
}

var sensitive = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}|https?://\S+|\bDE[0-9]{20}\b|\+?[0-9][0-9 ()./-]{10,}[0-9]`)

// Conservative additional guard; the complete v8 operational guard is retained in the workflow artifact.
var operational = regexp.MustCompile(`(?i)storn|widerruf|erstatt|rückzahl|ruckzahl|gutschrift|kulanz|preisnachlass|rabatt|garant|verbindlich|sp[äa]testens|spaetestens|sicher\s+(bis|am|morgen)|morgen|stopp|unterbrech|zurückstell|freigeb|umbuch|pausier|abbrech|lieferzusage|liefergarantie|priorisier|produktionsfreigabe|rückgängig|änder|verschieb|veranlass`)

func Validate(c Content, checks Checks, approval bool) error {
	if !contains(Types, c.Type) || !contains(Categories, c.Category) || !text(c.Topic, 150) || strings.TrimSpace(c.Topic) == "" || !text(c.Retrieval, 900) || utf8.RuneCountInString(strings.TrimSpace(c.Retrieval)) < 12 {
		return ErrInput
	}
	for _, s := range []string{c.Question, c.Answer, c.Rule, c.Evidence, c.Source, c.Limits} {
		if !text(s, 6000) {
			return ErrInput
		}
	}
	for _, s := range []string{c.Topic, c.Retrieval, c.Question, c.Answer, c.Rule, c.Evidence, c.Source, c.Limits} {
		if sensitive.MatchString(s) {
			return ErrInput
		}
	}
	if !approval {
		return nil
	}
	if !checks.Privacy || !checks.Content {
		return ErrInput
	}
	if c.Type == "factual_review" {
		return ErrConflict
	}
	combined := norm.NFKC.String(strings.Join([]string{c.Topic, c.Retrieval, c.Question, c.Answer, c.Rule}, " "))
	individualContext := regexp.MustCompile(`(?i)(ihre|deine|dieser|konkrete).{0,30}(bestellung|auftrag|sendung|lieferung|rechnung|produktion)`)
	individualState := regexp.MustCompile(`(?i)versendet|zugestellt|bearbeitet|gepr[üu]ft|erstattet|storniert`)
	fulfilmentSubject := regexp.MustCompile(`(?i)r[üu]cksendung|widerruf|r[üu]ckerstattung|reklamation|gutschrift|r[üu]ckzahlung`)
	fulfilmentPromise := regexp.MustCompile(`(?i)wir\s+(k[öo]nnen|werden|haben)|ich\s+(kann|werde|habe)|erfolgt|erhalten|gew[äa]hren|erstatten`)
	unsafeIndividual := individualContext.MatchString(combined) && individualState.MatchString(combined)
	unsafeFulfilment := fulfilmentSubject.MatchString(combined) && fulfilmentPromise.MatchString(combined)
	if (c.Type == "employee_fact" || c.Type == "response_example" || c.Type == "style_rule") && (operational.MatchString(combined) || unsafeIndividual || unsafeFulfilment) {
		return ErrInput
	}
	switch c.Type {
	case "employee_fact":
		if !checks.General || len(strings.TrimSpace(c.Evidence)) < 8 || strings.TrimSpace(c.Source) == "" || strings.TrimSpace(c.Limits) == "" || len(strings.TrimSpace(c.Rule)) < 8 || !strings.Contains(strings.ToLower(c.Evidence), strings.ToLower(c.Rule)) {
			return ErrInput
		}
	case "response_example":
		if !checks.StyleOnly || len(strings.TrimSpace(c.Answer)) < 25 {
			return ErrInput
		}
	case "style_rule":
		if !checks.StyleOnly || len(strings.TrimSpace(c.Rule)) < 8 {
			return ErrInput
		}
	case "classification_example":
		if !checks.Correction || len(c.CorrectedCategories) == 0 || strings.TrimSpace(c.Evidence) == "" || len(c.OriginalCategories) == 0 {
			return ErrInput
		}
		changed := false
		for _, cat := range c.CorrectedCategories {
			if !contains(Categories, cat) {
				return ErrInput
			}
			if !contains(c.OriginalCategories, cat) {
				changed = true
			}
		}
		if !changed {
			return ErrInput
		}
	}
	return nil
}
func ValidVector(v []float64) bool {
	if len(v) != 1536 {
		return false
	}
	nonzero := false
	for _, n := range v {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return false
		}
		nonzero = nonzero || n != 0
	}
	return nonzero
}
