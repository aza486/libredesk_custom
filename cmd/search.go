package main

import (
	"fmt"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength   = 3
	maxConversationResults = 100
	maxMessageResults      = 50
)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Conversations(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, c := range results {
		uuids[i] = c.UUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.ConversationResult, 0, len(allowed))
	for _, c := range results {
		if _, ok := set[c.UUID]; ok {
			out = append(out, c)
		}
	}
		if len(out) > maxConversationResults {
		out = out[:maxConversationResults]
	}
	return r.SendEnvelope(out)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app, user, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Messages(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, m := range results {
		uuids[i] = m.ConversationUUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.MessageResult, 0, len(allowed))
	for _, m := range results {
		if _, ok := set[m.ConversationUUID]; ok {
			out = append(out, m)
		}
	}
	if len(out) > maxMessageResults {
		out = out[:maxMessageResults]
	}
	return r.SendEnvelope(out)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app, auser, q, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Contacts(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	viewer, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	out := make([]smodels.ContactResult, 0, len(results))
	for _, contact := range results {
		allowed, err := app.user.CanAccessContact(contact.ID, viewer)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if allowed {
			out = append(out, contact)
		}
	}
	return r.SendEnvelope(out)
}

func searchInputs(r *fastglue.Request) (*App, amodels.User, string, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	q := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(q) < minSearchQueryLength {
		return app, user, "", envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	return app, user, q, nil
}

func uuidSet(uuids []string) map[string]struct{} {
	s := make(map[string]struct{}, len(uuids))
	for _, u := range uuids {
		s[u] = struct{}{}
	}
	return s
}
