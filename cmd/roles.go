package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetRoles returns all roles
func handleGetRoles(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.role.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetRole returns a single role
func handleGetRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	role, err := app.role.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(role)
}

// handleDeleteRole deletes a role
func handleDeleteRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err := app.role.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Role deleted successfully")
}

// handleCreateRole creates a new role
func handleCreateRole(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}
	if err := app.role.Create(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Role created successfully")
}

// handleUpdateRole updates a role
func handleUpdateRole(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		req   = models.Role{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}
	if err := app.role.Update(id, req);err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Role updated successfully")
}
