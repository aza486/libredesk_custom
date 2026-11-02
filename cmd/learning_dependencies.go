package main

import (
	"context"
	"errors"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

func learningDependencyFailure(service string, status int, err error) error {
	kind := "request_failed"
	var upstream *aiassist.HTTPStatusError
	switch {
	case errors.As(err, &upstream):
		kind = "http_status"
		status = upstream.Status
	case errors.Is(err, aiassist.ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		kind = "timeout"
	case errors.Is(err, aiassist.ErrInvalidResponse):
		kind = "invalid_response"
	case status == 400 || status == 422:
		kind = "request_rejected"
	}
	return &learning.DependencyError{Service: service, Kind: kind, Status: status}
}

// Never log err.Error(), pq.Message/Detail/Hint, HTTP bodies, DSNs or input values.
func learningFailureFields(stage string, err error) []any {
	switch stage {
	case "database_open", "database_connect", "embedding_config", "anonymizer_config", "review_config", "review_request", "review_validation", "publish":
	default:
		stage = "internal"
	}
	fields := []any{"stage", stage}
	var dependency *learning.DependencyError
	var database *pq.Error
	switch {
	case learning.DiagnosticCode(err) != "":
		fields = append(fields, "cause", learning.DiagnosticCode(err))
	case errors.As(err, &dependency):
		service := "dependency"
		if dependency.Service == "anonymizer" || dependency.Service == "embedding" {
			service = dependency.Service
		}
		kind := "unavailable"
		switch dependency.Kind {
		case "not_configured", "timeout", "invalid_response", "request_failed", "request_rejected", "privacy_rejected", "http_status":
			kind = dependency.Kind
		}
		fields = append(fields, "cause", service+"_"+kind)
		if dependency.Status >= 100 && dependency.Status <= 599 {
			fields = append(fields, "upstream_status", dependency.Status)
		}
	case errors.Is(err, aiassist.ErrInvalidConfiguration):
		fields = append(fields, "cause", "configuration_invalid")
	case errors.As(err, &database):
		code := string(database.Code)
		valid := len(code) == 5
		for _, c := range code {
			if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z') {
				valid = false
			}
		}
		fields = append(fields, "cause", "database")
		if valid {
			fields = append(fields, "sqlstate", code)
		}
	case errors.Is(err, context.DeadlineExceeded):
		fields = append(fields, "cause", "deadline_exceeded")
	case errors.Is(err, context.Canceled):
		fields = append(fields, "cause", "request_canceled")
	case errors.Is(err, learning.ErrUnavailable):
		fields = append(fields, "cause", "learning_not_available")
	default:
		fields = append(fields, "cause", "internal_failure")
	}
	return fields
}
func reportLearningFailure(lo *logf.Logger, stage string, err error) {
	if lo != nil {
		lo.Error("AI learning operation failed", learningFailureFields(stage, err)...)
	}
}
