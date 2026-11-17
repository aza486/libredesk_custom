package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/zerodha/logf"
)

func TestCasePublisherExplicitConfiguration(t *testing.T) {
	embed := learning.Embedder(func(context.Context, string) ([]float64, error) { return nil, nil })
	for _, tt := range []struct {
		enabled bool
		dsn     string
		embed   learning.Embedder
		valid   bool
	}{
		{false, "", nil, true}, {false, "invalid unused DSN", nil, true}, {true, "", embed, false}, {true, "postgres://SENSITIVE", nil, false}, {true, "postgres://isolated-publisher", embed, true},
	} {
		err := validateCasePublisherConfig(tt.enabled, tt.dsn, tt.embed)
		if (err == nil) != tt.valid {
			t.Fatal("configuration gate", tt.enabled, tt.valid, err)
		}
		if err != nil && strings.Contains(err.Error(), "SENSITIVE") {
			t.Fatal("DSN exposed")
		}
	}
}

func TestCasePublisherDiagnosticsContainOnlyCodesAndUUIDs(t *testing.T) {
	const id = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	for _, tt := range []struct{ code, id string }{
		{"case_database_failed", id}, {"case_embedding_failed", id}, {"case_revision_conflict", id}, {"case_validation_failed", id}, {"case_vector_invalid", id}, {"SENSITIVE_CUSTOMER_TEXT", "SENSITIVE_EMBEDDING"},
	} {
		var output bytes.Buffer
		lo := logf.New(logf.Opts{Writer: &output})
		reportCasePublisherFailure(&lo, &learning.CasePublishError{Code: tt.code, ID: tt.id})
		if strings.Contains(output.String(), "SENSITIVE") {
			t.Fatal("content exposed", output.String())
		}
		if !strings.Contains(output.String(), "case_publish") {
			t.Fatal("missing stage")
		}
		if tt.id == id && (!strings.Contains(output.String(), id) || !strings.Contains(output.String(), tt.code)) {
			t.Fatal("missing safe diagnosis")
		}
	}
}

type learningEmbeddingProbe struct{ t *testing.T }

func (p learningEmbeddingProbe) Call(_ context.Context, header, id string, payload, result any) (int, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		p.t.Fatal(err)
	}
	var request struct {
		Model      string `json:"model"`
		Dimensions int    `json:"dimensions"`
		Input      string `json:"input"`
	}
	if err = json.Unmarshal(raw, &request); err != nil {
		p.t.Fatal(err)
	}
	if request.Model != "text-embedding-3-small" || request.Dimensions != 1536 || request.Input != "Geprüfter Suchtext" || header != "Authorization" || id != "" {
		p.t.Fatal("embedding contract", request, header)
	}
	var fields map[string]any
	json.Unmarshal(raw, &fields)
	if len(fields) != 3 {
		p.t.Fatal("additional case data in embedding request")
	}
	vector := make([]float64, 1536)
	vector[0] = 1
	body, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"embedding": vector}}})
	return 200, json.Unmarshal(body, result)
}
func TestCaseAndFactPublishersShareEmbeddingContract(t *testing.T) {
	embed := newLearningEmbedder(learningEmbeddingProbe{t})
	vector, err := embed(context.Background(), "Geprüfter Suchtext")
	if err != nil || !learning.ValidVector(vector) {
		t.Fatal("invalid embedding result", err)
	}
}
