package main

import (
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

// Scope is part of every route. Definition updates cannot change it, and IDs
// from another scope are rejected by the manager rather than looked up globally.
func handleSidebarFolders(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	scope := r.RequestCtx.UserValue("scope").(string)
	scopeID, err := strconv.Atoi(r.RequestCtx.UserValue("scopeID").(string))
	invalid := func() error { return r.SendErrorEnvelope(400, "Ungültige Ordneranfrage", nil, envelope.InputError) }
	if err != nil || scopeID < 0 || scopeID > 2147483647 {
		return invalid()
	}
	path := string(r.RequestCtx.Path())
	id := 0
	if value, ok := r.RequestCtx.UserValue("id").(string); ok {
		id, err = strconv.Atoi(value)
		if err != nil || id <= 0 {
			return invalid()
		}
	}
	if strings.HasSuffix(path, "/counts") {
		counts, err := app.conversation.SidebarFolderCounts(user.ID, scope, scopeID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(counts)
	}
	if strings.HasSuffix(path, "/conversations") {
		page, size := getPagination(r)
		args := r.RequestCtx.QueryArgs()
		rows, err := app.conversation.GetSidebarFolderConversations(user.ID, scope, scopeID, id, string(args.Peek("order")), string(args.Peek("order_by")), page, size)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		total := 0
		if len(rows) > 0 {
			total = rows[0].Total
		} else if page > 1 {
			counts, e := app.conversation.SidebarFolderCounts(user.ID, scope, scopeID)
			if e != nil {
				return sendErrorEnvelope(r, e)
			}
			total = counts[id]
		}
		return r.SendEnvelope(envelope.PageResults{Results: rows, Total: total, Page: page, PerPage: size, TotalPages: (total + size - 1) / size})
	}
	method := string(r.RequestCtx.Method())
	if method == "GET" {
		folders, err := app.conversation.ListSidebarFolders(user.ID, scope, scopeID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(folders)
	}
	action := ""
	folder := conversation.SidebarFolder{ShowCount: true}
	var ids []int
	switch method {
	case "POST":
		action = "create"
	case "PUT":
		action = "update"
	case "DELETE":
		action = "delete"
	}
	if strings.HasSuffix(path, "/order") {
		var req struct {
			IDs []int `json:"ids"`
		}
		if err = r.Decode(&req, "json"); err != nil {
			return invalid()
		}
		ids = req.IDs
		action = "reorder"
	} else if action == "create" || action == "update" {
		if err = r.Decode(&folder, "json"); err != nil {
			return invalid()
		}
		if (folder.ScopeType != "" && folder.ScopeType != scope) || (folder.ScopeID != 0 && folder.ScopeID != scopeID) {
			return invalid()
		}
	}
	folders, err := app.conversation.MutateSidebarFolder(user.ID, scope, scopeID, id, action, folder, ids)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(folders)
}
