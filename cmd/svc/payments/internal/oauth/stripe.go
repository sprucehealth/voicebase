package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	oauthSiteHost        = `connect.stripe.com`
	oauthDeauthorizePath = `/oauth/deauthorize`
	oauthDeauthorizeURL  = `https://` + oauthSiteHost + oauthDeauthorizePath
)

func stripeError(err error) error {
	return fmt.Errorf("Stripe Error: %s", err)
}

// StripeOAuth described the interface for raw stripe oauth interaction using net/http
type StripeOAuth interface {
	DisconnectStripeAccount(userID string) error
	RequestStripeAccessToken(code string) (*StripeAccessTokenResponse, error)
}

type stripeOAuth struct {
	stripeSecretKey string
	stripeClientID  string
}

// NewStripe returns and instance of StripeHTTP
func NewStripe(stripeSecretKey, stripeClientID string) StripeOAuth {
	return &stripeOAuth{
		stripeSecretKey: stripeSecretKey,
		stripeClientID:  stripeClientID,
	}
}

type disconnectStripeAccountResponse struct {
	UserID           string `json:"stripe_user_id"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (s *stripeOAuth) DisconnectStripeAccount(userID string) error {
	body, err := s.makeRequest(&url.URL{
		Scheme: "https",
		Host:   oauthSiteHost,
		Path:   oauthDeauthorizePath,
	}, url.Values{
		`client_secret`:  []string{s.stripeSecretKey},
		`client_id`:      []string{s.stripeClientID},
		`stripe_user_id`: []string{userID},
	})
	if err != nil {
		return errors.Trace(err)
	}

	disconnectResponse := &disconnectStripeAccountResponse{}
	if err := json.NewDecoder(body).Decode(disconnectResponse); err != nil {
		return errors.Trace(err)
	}
	if disconnectResponse.Error != "" {
		return errors.Trace(stripeError(fmt.Errorf("%s: %s", disconnectResponse.Error, disconnectResponse.ErrorDescription)))
	}
	golog.Debugf("Disconnected stripe account %s", disconnectResponse.UserID)

	return nil
}

const (
	oauthTokenPath = `/oauth/token`
	oauthTokenURL  = `https://` + oauthSiteHost + oauthTokenPath
)

// StripeAccessTokenResponse represents the response from a stripe access token request
type StripeAccessTokenResponse struct {
	AccessToken          string `json:"access_token"`
	LiveMode             bool   `json:"livemode"`
	RefreshToken         string `json:"refresh_token"`
	TokenType            string `json:"token_type"`
	StripePublishableKey string `json:"stripe_publishable_key"`
	StripeUserID         string `json:"stripe_user_id"`
	Scope                string `json:"scope"`
	Error                string `json:"error"`
	ErrorDescription     string `json:"error_description"`
}

func (s *stripeOAuth) RequestStripeAccessToken(code string) (*StripeAccessTokenResponse, error) {
	body, err := s.makeRequest(&url.URL{
		Scheme: "https",
		Host:   oauthSiteHost,
		Path:   oauthTokenPath,
	}, url.Values{
		`client_secret`: []string{s.stripeSecretKey},
		`code`:          []string{code},
		`grant_type`:    []string{`authorization_code`},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	accessTokenResp := &StripeAccessTokenResponse{}
	if err := json.NewDecoder(body).Decode(accessTokenResp); err != nil {
		return nil, errors.Trace(err)
	}
	if accessTokenResp.Error != "" {
		return nil, errors.Trace(stripeError(fmt.Errorf("%s: %s", accessTokenResp.Error, accessTokenResp.ErrorDescription)))
	}

	return accessTokenResp, nil
}

func (s *stripeOAuth) makeRequest(u *url.URL, v url.Values) (io.Reader, error) {
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(v.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	ball, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("Error reading body response code %d from %s - %s", resp.StatusCode, oauthDeauthorizeURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Non 200 Response %d from %s - %s - Body: %s", resp.StatusCode, oauthDeauthorizeURL, err, string(ball))
	}

	return bytes.NewReader(ball), nil
}
