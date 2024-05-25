package main

import (
	"encoding/json"
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleUpsertConvTag(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		p       = r.RequestCtx.PostArgs()
		uuid    = r.RequestCtx.UserValue("uuid").(string)
		tagJSON = p.Peek("tag_ids")
		tagIDs  = []int{}
	)
	err := json.Unmarshal(tagJSON, &tagIDs)
	if err != nil {
		app.lo.Error("unmarshalling tag ids", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	if err := app.tags.UpsertConvTag(uuid, tagIDs); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope("ok")
}

func handleGetTags(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.tags.GetAllTags()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}
	return r.SendEnvelope(t)
}
