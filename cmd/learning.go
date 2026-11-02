package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func initLearning(ctx context.Context, lo *logf.Logger) *learning.Manager {
	dsn := ko.String("ai_learning.database_dsn")
	if dsn == "" {
		return nil
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		logLearningFatal(lo, "database_open", err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	// No schema initialization and no implicit production migration.
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = db.PingContext(timeout); err != nil {
		logLearningFatal(lo, "database_connect", err)
	}
	m := &learning.Manager{DB: db, ReportFailure: func(stage string, err error) { reportLearningFailure(lo, stage, err) }}
	if ko.String("ai_learning.embedding_endpoint") != "" {
		client, e := aiassist.New(ko.String("ai_learning.embedding_endpoint"), ko.String("ai_learning.embedding_token"))
		if e != nil {
			logLearningFatal(lo, "embedding_config", e)
		}
		m.Embed = func(ctx context.Context, text string) ([]float64, error) {
			var result struct {
				Data []struct {
					Embedding []float64 `json:"embedding"`
				} `json:"data"`
			}
			status, err := client.Call(ctx, "Authorization", "", map[string]any{"model": "text-embedding-3-small", "dimensions": 1536, "input": text}, &result)
			if err != nil {
				return nil, learningDependencyFailure("embedding", status, err)
			}
			if len(result.Data) != 1 {
				return nil, &learning.DependencyError{Service: "embedding", Kind: "invalid_response"}
			}
			return result.Data[0].Embedding, nil
		}
	}
	m.Anonymize, err = newLearningAnonymizer(aiassist.Config{
		Endpoint: ko.String("ai_learning.anonymizer_endpoint"), Token: ko.String("ai_learning.anonymizer_token"),
		InternalHTTPEndpoint: ko.String("ai_learning.anonymizer_internal_http_endpoint"), InternalHTTPIP: ko.String("ai_learning.anonymizer_internal_http_ip"),
	})
	if err != nil {
		logLearningFatal(lo, "anonymizer_config", err)
	}
	if m.Anonymize == nil {
		reportLearningFailure(lo, "anonymizer_config", &learning.DependencyError{Service: "anonymizer", Kind: "not_configured"})
	}

	go m.Run(ctx)
	return m
}
func logLearningFatal(lo *logf.Logger, stage string, err error) {
	reportLearningFailure(lo, stage, err)
	panic("invalid or unavailable ai_learning configuration; credentials withheld")
}
func handleLearning(r *fastglue.Request) error {
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	app := r.Context.(*App)
	caller, ok := r.RequestCtx.UserValue("user").(amodels.User)
	if !ok || r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return r.SendErrorEnvelope(403, "Mitarbeiter-Sitzung erforderlich.", nil, envelope.PermissionError)
	}
	user, err := app.user.GetAgentCachedOrLoad(caller.ID)
	if err != nil {
		return r.SendErrorEnvelope(403, "Aktiver Mitarbeiter erforderlich.", nil, envelope.PermissionError)
	}
	if !user.Enabled || user.Type != "agent" || user.IsSystemUser() || !learningAllowed(user.HasAdminRole(), user.Permissions) {
		return r.SendErrorEnvelope(403, "Keine Berechtigung für KI-Lernverwaltung.", nil, envelope.PermissionError)
	}
	if app.learning == nil {
		reportLearningFailure(app.lo, "review_config", learning.ErrUnavailable)
		return r.SendErrorEnvelope(503, "KI-Lernverwaltung ist noch nicht konfiguriert.", nil, envelope.GeneralError)
	}
	ctx, cancel := context.WithTimeout(app.ctx, 40*time.Second)
	defer cancel()
	// Copy: authorization and context belong to this request, never a global user store.
	view := *app.learning
	view.View = learningView(app, user)
	id, _ := r.RequestCtx.UserValue("id").(string)
	if string(r.RequestCtx.Method()) == "GET" {
		if id != "" {
			item, e := view.Get(ctx, id)
			return learningResult(r, item, e)
		}
		args := r.RequestCtx.QueryArgs()
		page := 1
		if args.Has("page") {
			page, err = strconv.Atoi(string(args.Peek("page")))
			if err != nil {
				return learningResult(r, nil, learning.ErrInput)
			}
		}
		result, e := view.List(ctx, learning.Filter{Tab: string(args.Peek("tab")), Type: string(args.Peek("type")), Status: string(args.Peek("status")), Page: page})
		return learningResult(r, result, e)
	}
	var req learning.Decision
	if err = decodeLearningDecision(r.RequestCtx.PostBody(), &req); err != nil {
		return learningResult(r, nil, err)
	}
	result, e := view.Decide(ctx, id, strconv.Itoa(user.ID), req)
	return learningResult(r, map[string]string{"id": result}, e)
}
func learningAllowed(admin bool, permissions []string) bool {
	if admin {
		return true
	}
	for _, p := range permissions {
		if p == "ai_learning:manage" {
			return true
		}
	}
	return false
}
func learningResult(r *fastglue.Request, data any, err error) error {
	switch {
	case err == nil:
		return r.SendEnvelope(data)
	case errors.Is(err, learning.ErrForbidden):
		return r.SendErrorEnvelope(403, "Kein Zugriff auf die zugrunde liegende Konversation.", nil, envelope.PermissionError)
	case errors.Is(err, learning.ErrInput):
		return r.SendErrorEnvelope(400, "Vorschlag kann nicht sicher übernommen werden.", nil, envelope.InputError)
	case errors.Is(err, learning.ErrConflict):
		return r.SendErrorEnvelope(409, "Eintrag geändert oder Aktion nicht zulässig. Bitte neu laden.", nil, envelope.InputError)
	case errors.Is(err, sql.ErrNoRows):
		return r.SendErrorEnvelope(404, "Lerninhalt nicht gefunden.", nil, envelope.InputError)
	default:
		if app, ok := r.Context.(*App); ok {
			reportLearningFailure(app.lo, "review_request", err)
		}
		return r.SendErrorEnvelope(503, "Lernverwaltung vorübergehend nicht verfügbar. Keine Freigabe bestätigt.", nil, envelope.GeneralError)
	}
}

func decodeLearningDecision(body []byte, req *learning.Decision) error {
	if !utf8.Valid(body) || len(body) > 1024 || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		return learning.ErrInput
	}
	var wire struct {
		Action   string `json:"action"`
		Revision *int64 `json:"revision"`
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&wire) != nil || d.Decode(new(any)) != io.EOF || wire.Revision == nil || *wire.Revision < 0 {
		return learning.ErrInput
	}
	req.Action = wire.Action
	req.Revision = *wire.Revision
	return nil
}

func learningView(app *App, user umodels.User) func(context.Context, *learning.Item) error {
	return func(_ context.Context, item *learning.Item) error {
		var refs struct {
			Original string `json:"original_message_uuid"`
			Outgoing string `json:"outgoing_message_uuid"`
		}
		if json.Unmarshal(item.Provenance, &refs) != nil {
			return learning.ErrForbidden
		}
		if _, e := uuid.Parse(refs.Original); e != nil {
			return learning.ErrForbidden
		}
		if _, e := uuid.Parse(refs.Outgoing); e != nil {
			return learning.ErrForbidden
		}
		if app.conversation == nil || app.authz == nil {
			return learning.ErrForbidden
		}
		original, e := app.conversation.GetMessage(refs.Original)
		if e != nil || original.Private || original.Type != "incoming" || original.SenderType != "contact" {
			return learning.ErrForbidden
		}
		if _, e = enforceConversationAccess(app, original.ConversationUUID, user); e != nil {
			return learning.ErrForbidden
		}
		outgoing, e := app.conversation.GetMessage(refs.Outgoing)
		if e != nil || outgoing.Private || outgoing.Type != "outgoing" || outgoing.SenderType != "agent" || outgoing.Status != "sent" || outgoing.ConversationUUID != original.ConversationUUID {
			return learning.ErrForbidden
		}
		var originalMeta, outgoingMeta map[string]json.RawMessage
		if json.Unmarshal(original.Meta, &originalMeta) != nil || json.Unmarshal(outgoing.Meta, &outgoingMeta) != nil {
			return learning.ErrForbidden
		}
		if _, deleted := originalMeta["deleted_at"]; deleted {
			return learning.ErrForbidden
		}
		if _, deleted := outgoingMeta["deleted_at"]; deleted {
			return learning.ErrForbidden
		}
		for _, key := range []string{"reply_to_message_uuid", "ai_feedback_original_message_uuid"} {
			if raw, exists := outgoingMeta[key]; exists {
				var source string
				if json.Unmarshal(raw, &source) != nil || source != refs.Original {
					return learning.ErrForbidden
				}
			}
		}
		item.Context = &learning.DisplayContext{Question: stringutil.HTML2Text(original.Content), Answer: stringutil.HTML2Text(item.SentAnswer)}
		item.SentAnswer = ""
		return nil
	}
}
