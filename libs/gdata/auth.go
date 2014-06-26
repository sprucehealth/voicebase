package gdata

import (
	"time"

	"github.com/sprucehealth/backend/third_party/code.google.com/p/goauth2/oauth"
)

const (
	SpreadsheetScope = "https://spreadsheets.google.com/feeds"
	AuthURL          = "https://accounts.google.com/o/oauth2/auth"
	TokenURL         = "https://accounts.google.com/o/oauth2/token"
	RedirectURL      = "urn:ietf:wg:oauth:2.0:oob"
)

func MakeOauthTransport(scope, clientId, clientSecret, accessToken, refreshToken string) *oauth.Transport {
	return &oauth.Transport{
		Config: &oauth.Config{
			ClientId:     clientId,
			ClientSecret: clientSecret,
			Scope:        scope,
			AuthURL:      AuthURL,
			TokenURL:     TokenURL,
			RedirectURL:  RedirectURL,
		},
		Token: &oauth.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       time.Time{}, // no expiry
		},
		Transport: nil,
	}
}
