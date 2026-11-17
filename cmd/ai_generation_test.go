package main

import (
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"strings"
	"testing"
)

func TestGenerateReplyOptionalParameters(t *testing.T) {
	for _, tc := range []struct {
		name  string
		req   generateReplyReq
		valid bool
	}{
		{"legacy empty", generateReplyReq{}, true},
		{"legacy draft instruction", generateReplyReq{ConversationUUID: "ticket", Instruction: "existing draft"}, true},
		{"new", generateReplyReq{ConversationUUID: "ticket", Instruction: "thank customer", Mode: "generate"}, true},
		{"edit", generateReplyReq{ConversationUUID: "ticket", Instruction: "correct spelling", Mode: "edit", DraftHTML: "<p>Draft</p>"}, true},
		{"translate", generateReplyReq{ConversationUUID: "ticket", Instruction: "translate into English", Mode: "edit", DraftHTML: "<p>Hallo</p>"}, true},
		{"empty HTML edit", generateReplyReq{ConversationUUID: "ticket", Instruction: "translate", Mode: "edit", DraftHTML: "<p></p>"}, false},
		{"unknown mode", generateReplyReq{Mode: "send"}, false},
		{"edit no context", generateReplyReq{Mode: "edit", Instruction: "translate", DraftHTML: "draft"}, false},
		{"edit no instruction", generateReplyReq{ConversationUUID: "ticket", Mode: "edit", DraftHTML: "draft"}, false},
		{"unexpected draft", generateReplyReq{Mode: "generate", DraftHTML: "draft"}, false},
		{"NUL", generateReplyReq{Instruction: "a\x00b"}, false},
		{"invalid UTF8", generateReplyReq{Instruction: string([]byte{0xff})}, false},
		{"Unicode limit", generateReplyReq{Instruction: strings.Repeat("😀", 8000)}, true},
		{"instruction too long", generateReplyReq{ConversationUUID: "ticket", Mode: "generate", Instruction: strings.Repeat("x", 8001)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if validGenerateReply(tc.req) != tc.valid {
				t.Fatalf("validation mismatch")
			}
		})
	}
}

func TestTranscriptionRequiresSession(t *testing.T) {
	for _, method := range []string{"", authMethodAPIKey} {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("auth_method", method)
		if err := handleAITranscribe(&fastglue.Request{RequestCtx: ctx, Context: &App{}}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != 403 {
			t.Fatal("accepted non-session audio")
		}
	}
}
