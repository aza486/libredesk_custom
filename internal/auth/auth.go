package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/stringutil"
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

type Provider struct {
	ID           int
	Provider     string
	ProviderURL  string
	RedirectURL  string
	ClientID     string
	ClientSecret string
}

type Config struct {
	Providers []Provider
}

type Auth struct {
	mu        sync.RWMutex
	cfg       Config
	oauthCfgs map[int]oauth2.Config
	verifiers map[int]*oidc.IDTokenVerifier
	sess      *simplesessions.Manager
	logger    *logf.Logger
	rd        *redis.Client
}

// New initializes an OIDC configuration for multiple providers.
func New(cfg Config, rd *redis.Client, logger *logf.Logger) (*Auth, error) {
	oauthCfgs := make(map[int]oauth2.Config)
	verifiers := make(map[int]*oidc.IDTokenVerifier)

	for _, provider := range cfg.Providers {
		oidcProv, err := oidc.NewProvider(context.Background(), provider.ProviderURL)
		if err != nil {
			logger.Error("error initializing oidc provider", "error", err, "provider", provider.Provider)
			continue
		}

		oauthCfg := oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			Endpoint:     oidcProv.Endpoint(),
			RedirectURL:  provider.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		verifier := oidcProv.Verifier(&oidc.Config{ClientID: provider.ClientID})

		oauthCfgs[provider.ID] = oauthCfg
		verifiers[provider.ID] = verifier
	}

	sess := simplesessions.New(simplesessions.Options{
		EnableAutoCreate: true,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			IsHTTPOnly: true,
			IsSecure:   true,
			MaxAge:     time.Hour * 6,
		},
	})

	st := sessredisstore.New(context.TODO(), rd)
	sess.UseStore(st)
	sess.SetCookieHooks(simpleSessGetCookieCB, simpleSessSetCookieCB)

	return &Auth{
		cfg:       cfg,
		oauthCfgs: oauthCfgs,
		verifiers: verifiers,
		sess:      sess,
		logger:    logger,
		rd:        rd,
	}, nil
}

// Reload reloads the auth configuration.
func (a *Auth) Reload(cfg Config) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	oauthCfgs := make(map[int]oauth2.Config)
	verifiers := make(map[int]*oidc.IDTokenVerifier)

	for _, provider := range cfg.Providers {
		oidcProv, err := oidc.NewProvider(context.Background(), provider.ProviderURL)
		if err != nil {
			a.logger.Error("error initializing oidc provider", "error", err, "provider", provider.Provider)
			continue
		}

		oauthCfg := oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			Endpoint:     oidcProv.Endpoint(),
			RedirectURL:  provider.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		verifier := oidcProv.Verifier(&oidc.Config{ClientID: provider.ClientID})

		oauthCfgs[provider.ID] = oauthCfg
		verifiers[provider.ID] = verifier
	}

	a.cfg = cfg
	a.oauthCfgs = oauthCfgs
	a.verifiers = verifiers

	return nil
}

// LoginURL returns the login URL for the given provider.
func (a *Auth) LoginURL(providerID int, state string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	oauthCfg, ok := a.oauthCfgs[providerID]
	if !ok {
		return "", envelope.NewError(envelope.InputError, "Provider not found", nil)
	}
	return oauthCfg.AuthCodeURL(state), nil
}

// ExchangeOIDCToken takes an OIDC authorization code, validates it, and returns an OIDC token for subsequent auth.
func (a *Auth) ExchangeOIDCToken(ctx context.Context, providerID int, code, nonce string) (string, OIDCclaim, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	oauthCfg, ok := a.oauthCfgs[providerID]
	if !ok {
		return "", OIDCclaim{}, fmt.Errorf("invalid provider ID: %d", providerID)
	}

	verifier, ok := a.verifiers[providerID]
	if !ok {
		return "", OIDCclaim{}, fmt.Errorf("invalid provider ID: %d", providerID)
	}

	tk, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return "", OIDCclaim{}, fmt.Errorf("error exchanging token: %v", err)
	}

	// Extract the ID Token from OAuth2 token.
	rawIDTk, ok := tk.Extra("id_token").(string)
	if !ok {
		return "", OIDCclaim{}, errors.New("id_token missing")
	}

	// Parse and verify ID Token payload.
	idTk, err := verifier.Verify(ctx, rawIDTk)
	if err != nil {
		return "", OIDCclaim{}, fmt.Errorf("error verifying ID token: %v", err)
	}

	if idTk.Nonce != nonce {
		return "", OIDCclaim{}, fmt.Errorf("nonce token mismatch")
	}

	var claims OIDCclaim
	if err := idTk.Claims(&claims); err != nil {
		return "", OIDCclaim{}, errors.New("error getting user from OIDC")
	}
	return rawIDTk, claims, nil
}

// SaveSession creates and sets a session (post successful login/auth).
func (a *Auth) SaveSession(user models.User, r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.NewSession(r, r)
	if err != nil {
		a.logger.Error("error creating login session", "error", err)
		return err
	}

	if err := sess.SetMulti(map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	}); err != nil {
		a.logger.Error("error setting login session", "error", err)
		return err
	}
	return nil
}

// SetCSRFCookie sets the CSRF token in the response cookie if not already set.
func (a *Auth) SetCSRFCookie(r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	cookie := r.RequestCtx.Request.Header.Cookie("csrf_token")
	if cookie == nil {
		token, err := generateCSRFToken()
		if err != nil {
			return err
		}
		var csrfCookie fasthttp.Cookie
		csrfCookie.SetKey("csrf_token")
		csrfCookie.SetValue(token)
		csrfCookie.SetPath("/")
		csrfCookie.SetSecure(true)
		csrfCookie.SetHTTPOnly(false)
		r.RequestCtx.Response.Header.SetCookie(&csrfCookie)
		return nil
	}
	return nil
}

func (a *Auth) ValidateSession(r *fastglue.Request) (models.User, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err != nil {
		a.logger.Error("error acquiring session", "error", err)
		return models.User{}, err
	}

	sessVals, err := sess.GetMulti("id", "email", "first_name", "last_name")
	if err != nil {
		a.logger.Error("error fetching session variables", "error", err)
		return models.User{}, err
	}

	var (
		userID, _    = sess.Int(sessVals["id"], nil)
		email, _     = sess.String(sessVals["email"], nil)
		firstName, _ = sess.String(sessVals["first_name"], nil)
		lastName, _  = sess.String(sessVals["last_name"], nil)
	)

	return models.User{
		ID:        userID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

// DestroySession destroys session
func (a *Auth) DestroySession(r *fastglue.Request) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sess, err := a.sess.Acquire(r.RequestCtx, r, r)
	if err != nil {
		a.logger.Error("error acquiring session", "error", err)
		return err
	}
	if err := sess.Destroy(); err != nil {
		a.logger.Error("error clearing session", "error", err)
		return err
	}
	return nil
}

// generateCSRFToken creates a random base64 encoded str.
func generateCSRFToken() (string, error) {
	b, err := stringutil.RandomAlNumString(30)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(b)), nil
}

// getRequestCookie returns fashttp.Cookie for the given name.
func getRequestCookie(name string, r *fastglue.Request) (*fasthttp.Cookie, error) {
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
