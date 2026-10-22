package main

import "testing"

func TestNormalClientReplySourceCannotOverrideAutomaticAnchor(t *testing.T) {
	for _, legacy := range []string{"", "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", "foreign-or-malformed-source"} {
		req := messageReq{ReplyToMessageUUID: legacy, SenderType: "agent", Message: "Keep answer"}
		req.ignoreNormalReplySource("email")
		if req.ReplyToMessageUUID != "" || req.Message != "Keep answer" {
			t.Fatal("legacy normal source survived", req)
		}
	}
}
func TestExplicitReplySourceContractsRemainIntact(t *testing.T) {
	for _, kind := range []string{"ai", "ai_adapted", "human"} {
		req := messageReq{ReplyToMessageUUID: "explicit", SenderType: "agent", AIResponse: &responseProvenance{Source: kind, SourceUUID: "explicit"}}
		req.ignoreNormalReplySource("email")
		if req.ReplyToMessageUUID != "explicit" || req.AIResponse.SourceUUID != "explicit" {
			t.Fatal("explicit source changed")
		}
	}
	req := messageReq{ReplyToMessageUUID: "explicit", SenderType: "agent", AIFeedbackOriginalMessageUUID: "explicit"}
	req.ignoreNormalReplySource("email")
	if req.ReplyToMessageUUID != "explicit" {
		t.Fatal("feedback source changed")
	}
	for _, req := range []messageReq{{ReplyToMessageUUID: "invalid", Private: true, SenderType: "agent"}, {ReplyToMessageUUID: "invalid", SenderType: "contact"}} {
		req.ignoreNormalReplySource("email")
		if req.ReplyToMessageUUID != "invalid" {
			t.Fatal("non-reply validation bypassed")
		}
	}
}
