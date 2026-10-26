package main

import (
	"encoding/json"
	"strings"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestManualReviewRequest(t *testing.T) {
	body := strings.Replace(validAnalysisRequest, `"kind":"n8n_ai_analysis"`, `"kind":"n8n_manual_review"`, 1)
	body = strings.Replace(body, `,"suggested_message":"Hallo\nAntwort als Text"`, "", 1)
	req, err := decodeAIAnalysisRequest([]byte(body))
	if err != nil || req.AIAnalysis.Kind != "n8n_manual_review" || req.AIAnalysis.SuggestedMessage != "" {
		t.Fatalf("manual review: %+v %v", req, err)
	}
	for _, invalid := range []string{
		strings.Replace(body, `"version":1`, `"version":2`, 1),
		strings.Replace(body, `aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa`, "forged", 1),
		strings.Replace(body, `"private":true`, `"private":false`, 1),
		strings.Replace(body, `"version":1`, `"version":1,"unexpected":true`, 1),
		strings.Replace(body, `"version":1`, `"version":1,"suggested_message":"invented"`, 1),
	} {
		if _, err := decodeAIAnalysisRequest([]byte(invalid)); err == nil {
			t.Fatal("invalid manual note accepted")
		}
	}
}

const validAnalysisRequest = `{"private":true,"sender_type":"agent","message":"<b>KI-Analyse</b>","ai_analysis":{"kind":"n8n_ai_analysis","version":1,"original_message_uuid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa","suggested_message":"Hallo\nAntwort als Text"}}`

func TestAIAnalysisRequest(t *testing.T) {
	req, err := decodeAIAnalysisRequest([]byte(validAnalysisRequest))
	if err != nil || req.Message != "<b>KI-Analyse</b>" || req.AIAnalysis.SuggestedMessage != "Hallo\nAntwort als Text" {
		t.Fatalf("valid request: %+v %v", req, err)
	}
	for _, body := range []string{`{}`, `{"private":false}`, validAnalysisRequest + `{}`, `{"private":true,"sender_type":"agent","message":"test","meta":{"ai_analysis":{}}}`, `{"private":true,"sender_type":"contact","message":"test"}`} {
		if _, err := decodeAIAnalysisRequest([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(validAnalysisRequest), &data)
	for _, field := range []string{"meta", "type", "echo_id", "attachments", "sender_id"} {
		data[field] = "unauthorized"
		raw, _ := json.Marshal(data)
		if _, err := decodeAIAnalysisRequest(raw); err == nil {
			t.Fatalf("accepted field %s", field)
		}
		delete(data, field)
	}
	data["private"] = false
	raw, _ := json.Marshal(data)
	if _, err := decodeAIAnalysisRequest(raw); err == nil {
		t.Fatal("public note accepted")
	}
	// Existing request schema remains unchanged and cannot persist general meta.
	var normal messageReq
	if err := json.Unmarshal([]byte(`{"private":true,"sender_type":"agent","message":"ordinary note"}`), &normal); err != nil || normal.Message != "ordinary note" || !normal.Private {
		t.Fatal("normal message request changed")
	}
}

func TestAIAnalysisCallerAuthorization(t *testing.T) {
	for _, tc := range []struct {
		method  string
		id      int
		allowed []int
		want    bool
	}{
		{authMethodAPIKey, 42, []int{42}, true}, {authMethodSession, 42, []int{42}, false}, {authMethodAPIKey, 43, []int{42}, false}, {authMethodAPIKey, 42, nil, false}, {"", 42, []int{42}, false},
	} {
		if got := allowedAIAnalysisCaller(tc.method, tc.id, tc.allowed); got != tc.want {
			t.Fatalf("authorization %+v: %v", tc, got)
		}
	}
	for _, method := range []string{"", authMethodSession, authMethodAPIKey} {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("auth_method", method)
		ctx.SetUserValue("user", amodels.User{ID: 43})
		if err := handleCreateAIAnalysisNote(&fastglue.Request{RequestCtx: ctx, Context: &App{aiNoteAgentIDs: []int{42}}}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
			t.Fatal("unauthorized integration caller accepted")
		}
	}
}
