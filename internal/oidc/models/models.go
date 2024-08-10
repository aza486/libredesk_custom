package models

import (
	"strings"
	"time"
)

// OIDC represents an OpenID Connect configuration.
type OIDC struct {
	ID              int       `db:"id" json:"id"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	Name            string    `db:"-" json:"name"`
	ProviderLogoURL string    `db:"-" json:"logo_url"`
	ProviderURL     string    `db:"provider_url" json:"provider_url"`
	ClientID        string    `db:"client_id" json:"client_id"`
	ClientSecret    string    `db:"client_secret" json:"client_secret"`
	RedirectURI     string    `db:"redirect_uri" json:"redirect_uri"`
}

// ProviderInfo holds the name and logo of a provider.
type ProviderInfo struct {
	Name string
	Logo string
}

var providerMap = map[string]ProviderInfo{
	"accounts.google.com": {Name: "Google", Logo: "https://lh3.googleusercontent.com/COxitqgJr1sJnIDe8-jiKhxDx1FrYbtRHKJ9z_hELisAlapwE9LUPh6fcXIfb5vwpbMl4xl9H9TRFPc5NOO8Sb3VSgIBrfRYvW6cUA"},
	"microsoftonline.com": {Name: "Microsoft", Logo: "https://logo"},
	"github.com":          {Name: "Github", Logo: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTrg0GanJf8uJHY8yuvL6Vyk47iVTx-AchsAA&s"},
}

// SetProviderInfo adds provider name and logo to an OIDC model.
func (oidc *OIDC) SetProviderInfo() {
	for url, info := range providerMap {
		if strings.Contains(oidc.ProviderURL, url) {
			oidc.Name = info.Name
			oidc.ProviderLogoURL = info.Logo
			return
		}
	}
	oidc.Name = "Custom"
	oidc.ProviderLogoURL = "https://path_to_default_logo"
}
