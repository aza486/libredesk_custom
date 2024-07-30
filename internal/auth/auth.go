package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/abhinavxd/artemis/internal/user/models"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	sessredisstore "github.com/zerodha/simplesessions/stores/redis/v3"
	"github.com/zerodha/simplesessions/v3"

	"golang.org/x/oauth2"
)

type OIDCclaim struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Sub           string `json:"sub"`
	Picture       string `json:"picture"`
}

type OIDCConfig struct {
	Enabled      bool   `json:"enabled"`
	ProviderURL  string `json:"provider_url"`
	RedirectURL  string `json:"redirect_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type Config struct {
	OIDC OIDCConfig
}

type Auth struct {
	cfg      Config
	oauthCfg oauth2.Config
	verifier *oidc.IDTokenVerifier
	sess     *simplesessions.Manager
	lo       *logf.Logger
}

// Callbacks takes two callback functions required by simplesessions.
type Callbacks struct {
	SetCookie func(cookie *http.Cookie, w interface{}) error
	GetCookie func(name string, r interface{}) (*http.Cookie, error)
}

// New inits a OIDC configuration
func New(cfg Config, rd *redis.Client, logger *logf.Logger) (*Auth, error) {
	provider, err := oidc.NewProvider(context.Background(), cfg.OIDC.ProviderURL)
	if err != nil {
		return nil, err
	}

	oauthCfg := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.OIDC.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDC.ClientID})

	maxAge := time.Hour * 24
	if maxAge.Seconds() == 0 {
		maxAge = time.Hour * 12
	}
	sess := simplesessions.New(simplesessions.Options{
		EnableAutoCreate: true,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			IsHTTPOnly: true,
			IsSecure:   true,
			MaxAge:     maxAge,
		},
	})
	st := sessredisstore.New(context.TODO(), rd)
	sess.UseStore(st)
	sess.SetCookieHooks(simpleSessGetCookieCB, simpleSessSetCookieCB)

	return &Auth{
		cfg:      cfg,
		oauthCfg: oauthCfg,
		verifier: verifier,
		lo:       logger,
		sess:     sess,
	}, nil
}

// LoginURL
func (a *Auth) LoginURL(state string) string {
	return a.oauthCfg.AuthCodeURL(state)
}

// ExchangeOIDCToken takes an OIDC authorization code, validates it, and returns an OIDC token for subsequent auth.
func (a *Auth) ExchangeOIDCToken(ctx context.Context, code string) (string, OIDCclaim, error) {
	tk, err := a.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return "", OIDCclaim{}, fmt.Errorf("error exchanging token: %v", err)
	}

	// Extract the ID Token from OAuth2 token.
	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", OIDCclaim{}, errors.New("`id_token` missing")
	}

	// Parse and verify ID Token payload.
	idTk, err := a.verifier.Verify(ctx, rawIDTk)
	if err != nil {
		return "", OIDCclaim{}, fmt.Errorf("error verifying ID token: %v", err)
	}

	var claims OIDCclaim
	if err := idTk.Claims(&claims); err != nil {
		return "", OIDCclaim{}, errors.New("error getting user from OIDC")
	}
	return rawIDTk, claims, nil
}

// SaveSession creates and sets a session (post successful login/auth).
func (o *Auth) SaveSession(user models.User, r *fastglue.Request) error {
	sess, err := o.sess.NewSession(r.RequestCtx, r.RequestCtx)
	if err != nil {
		o.lo.Error("error creating login session", "error", err)
		return err
	}

	if err := sess.SetMulti(map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"team_id":    user.TeamID,
	}); err != nil {
		o.lo.Error("error setting login session", "error", err)
		return err
	}
	return nil
}

// ValidateSession validates session and returns the user.
func (o *Auth) ValidateSession(r *fastglue.Request) (models.User, error) {
	sess, err := o.sess.Acquire(r.RequestCtx, r.RequestCtx, r.RequestCtx)
	if err != nil {
		return models.User{}, err
	}

	// Get the session variables
	sessVals, err := sess.GetMulti("id", "email", "first_name", "last_name")
	if err != nil {
		return models.User{}, err
	}

	var (
		userID, _    = sess.Int(sessVals["id"], nil)
		email, _     = sess.String(sessVals["email"], nil)
		firstName, _ = sess.String(sessVals["first_name"], nil)
		lastName, _  = sess.String(sessVals["last_name"], nil)
	)

	// Logged in?
	if userID <= 0 {
		o.lo.Error("error fetching session", "error", err)
		return models.User{}, err
	}

	return models.User{
		ID:        userID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

// DestroySession destroys session
func (o *Auth) DestroySession(r *fastglue.Request) error {
	sess, err := o.sess.Acquire(r.RequestCtx, r, r)
	if err != nil {
		o.lo.Error("error acquiring session", "error", err)
		return err
	}
	if err := sess.Destroy(); err != nil {
		o.lo.Error("error clearing session", "error", err)
		return err
	}
	return nil
}

// getRequestCookie returns fashttp.Cookie for the given name.
func getRequestCookie(name string, r *fastglue.Request) (*fasthttp.Cookie, error) {
	// Cookie value.
	val := r.RequestCtx.Request.Header.Cookie(name)
	if len(val) == 0 {
		return nil, nil
	}

	c := fasthttp.AcquireCookie()
	if err := c.ParseBytes(val); err != nil {
		return nil, err
	}

	return c, nil
}

// simpleSessGetCookieCB is the simplessesions callback for retrieving the session cookie
// from a fastglue request.
func simpleSessGetCookieCB(name string, r interface{}) (*http.Cookie, error) {
	req, ok := r.(*fastglue.Request)
	if !ok {
		return nil, errors.New("session callback doesn't have fastglue.Request")
	}

	// Create fast http cookie and parse it from cookie bytes.
	c, err := getRequestCookie(name, req)
	if c == nil {
		if err == nil {
			return nil, http.ErrNoCookie
		} else {
			return nil, err
		}
	}

	// Convert fasthttp cookie to net http cookie.
	return &http.Cookie{
		Name:     name,
		Value:    string(c.Value()),
		Path:     string(c.Path()),
		Domain:   string(c.Domain()),
		Expires:  c.Expire(),
		Secure:   c.Secure(),
		HttpOnly: c.HTTPOnly(),
		SameSite: http.SameSite(c.SameSite()),
	}, nil
}

// simpleSessSetCookieCB is the simplessesions callback for setting the session cookie
// to a fastglue request.
func simpleSessSetCookieCB(c *http.Cookie, w interface{}) error {
	req, ok := w.(*fastglue.Request)
	if !ok {
		return errors.New("session callback doesn't have fastglue.Request")
	}

	fc := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(fc)

	fc.SetKey(c.Name)
	fc.SetValue(c.Value)
	fc.SetPath(c.Path)
	fc.SetDomain(c.Domain)
	fc.SetExpire(c.Expires)
	fc.SetSecure(c.Secure)
	fc.SetHTTPOnly(c.HttpOnly)
	fc.SetSameSite(fasthttp.CookieSameSite(c.SameSite))

	req.RequestCtx.Response.Header.SetCookie(fc)
	return nil
}
