package gdata

import (
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/code.google.com/p/goauth2/oauth"
)

const (
	SpreadsheetScope = "https://spreadsheets.google.com/feeds"
	AuthURL          = "https://accounts.google.com/o/oauth2/auth"
	TokenURL         = "https://accounts.google.com/o/oauth2/token"
	RedirectURL      = "urn:ietf:wg:oauth:2.0:oob"
)

func MakeOauthTransport(scope, clientID, clientSecret, accessToken, refreshToken string) *oauth.Transport {
	return &oauth.Transport{
		Config: &oauth.Config{
			ClientId:     clientID,
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
