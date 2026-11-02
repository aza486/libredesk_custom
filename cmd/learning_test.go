package main

import (
	"github.com/abhinavxd/libredesk/internal/learning"
	"testing"
)

func TestLearningAuthorizationAndInput(t *testing.T) {
	if learningAllowed(false, []string{"ai:manage"}) || learningAllowed(false, nil) {
		t.Fatal("ordinary employee permitted")
	}
	if !learningAllowed(true, nil) || !learningAllowed(false, []string{"ai_learning:manage"}) {
		t.Fatal("reviewer denied")
	}
	for _, body := range []string{`{"action":"approve","agent_id":1}`, `{"action":"approve"} {}`, `[]`, `{"checks":{"unknown":true}}`} {
		if decodeLearningDecision([]byte(body), new(learning.Decision)) == nil {
			t.Fatal("untrusted field accepted", body)
		}
	}
}
