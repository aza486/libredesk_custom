package main

import (
	"context"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"testing"
)

func TestLearningHandlerPostgresAuthorization(t *testing.T) {
	db := testutil.NewPersonalDB(t, "learning_auth")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	if e := migrations.V2_17_0(db, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e := migrations.V2_17_0(db, nil, nil); e != nil {
		t.Fatal(e)
	}
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin'; INSERT INTO roles(name,permissions) VALUES('Learning Reviewer',ARRAY['ai_learning:manage']); INSERT INTO user_roles(user_id,role_id) SELECT 102,id FROM roles WHERE name='Learning Reviewer';`)
	users, e := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if e != nil {
		t.Fatal(e)
	}
	app := &App{ctx: context.Background(), user: users, learning: &learning.Manager{}}
	for _, tt := range []struct {
		name   string
		agent  int
		method string
		want   int
	}{{"admin", 104, authMethodSession, 400}, {"delegated", 102, authMethodSession, 400}, {"agent", 101, authMethodSession, 403}, {"contact", 105, authMethodSession, 403}, {"key", 104, "api_key", 403}, {"anonymous", 0, "", 403}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod("POST")
			ctx.Request.SetBodyString(`{"agent_id":999,"action":"approve"}`)
			ctx.SetUserValue("auth_method", tt.method)
			ctx.SetUserValue("id", "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa")
			if tt.agent != 0 {
				ctx.SetUserValue("user", amodels.User{ID: tt.agent})
			}
			r := &fastglue.Request{RequestCtx: ctx, Context: app}
			if err := handleLearning(r); err != nil {
				t.Fatal(err)
			}
			if ctx.Response.StatusCode() != tt.want {
				t.Fatalf("got %d want %d", ctx.Response.StatusCode(), tt.want)
			}
		})
	}
}
