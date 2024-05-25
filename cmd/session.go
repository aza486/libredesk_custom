package main

import (
	"errors"
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

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
