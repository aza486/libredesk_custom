package main

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		q   = string(r.RequestCtx.QueryArgs().Peek("query"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Query length should be at least 3 characters", nil))
	}

	conversations, err := app.search.Conversations(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(conversations)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		q   = string(r.RequestCtx.QueryArgs().Peek("query"))
	)

	if len(q) < minSearchQueryLength {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Query length should be at least 3 characters", nil))
	}

	messages, err := app.search.Messages(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(messages)
}
