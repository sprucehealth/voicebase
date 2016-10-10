package google

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

// AuthProvider returns an auth provider backed by LDAP
type AuthProvider struct{}

// NewAuthenticationProvider returns a Google Auth compatible authentication provider
func NewAuthenticationProvider() *AuthProvider {
	return &AuthProvider{}
}

// Authenticate checkes the validitidy of an id token and asserts the domain
func (ap *AuthProvider) Authenticate(ctx context.Context, idToken string) (string, error) {
	golog.ContextLogger(ctx).Debugf("Checking verification of id token %s", idToken)
	vResp, err := validateIDToken(idToken)
	if err != nil {
		return "", errors.Trace(err)
	}
	return vResp.Name, nil
}

// ErrForbidden represents that the provided account is forbidden
var ErrForbidden = errors.New("Forbidden")

const (
	oauthSiteHost      = "www.googleapis.com"
	oauthTokenInfoPath = "/oauth2/v3/tokeninfo"
)

type googleTokenValidationResponse struct {
	// These six fields are included in all Google ID Tokens.
	ISS string `json:"iss"`
	SUB string `json:"sub"`
	AZP string `json:"azp"`
	AUD string `json:"aud"`
	IAT string `json:"iat"`
	EXP string `json:"exp"`

	// These seven fields are only included when the user has granted the "profile" and
	// "email" OAuth scopes to the application.
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
}

func validateIDToken(idToken string) (*googleTokenValidationResponse, error) {
	body, err := makeRequest(&url.URL{
		Scheme: "https",
		Host:   oauthSiteHost,
		Path:   oauthTokenInfoPath,
	}, url.Values{
		`id_token`: []string{idToken},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	verificationResp := &googleTokenValidationResponse{}
	if err := json.NewDecoder(body).Decode(verificationResp); err != nil {
		return nil, errors.Trace(err)
	}

	// TODO: Is there a way to control this from the google app?
	if !strings.HasSuffix(verificationResp.Email, "@sprucehealth.com") {
		golog.Debugf("Only spruce health accounts allowed. Got %s", verificationResp.Email)
		return nil, ErrForbidden
	}

	return verificationResp, nil
}

func makeRequest(u *url.URL, v url.Values) (io.Reader, error) {
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(v.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	ball, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("Error reading body response code %d from %s - %s", resp.StatusCode, u.String(), err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Non 200 Response %d from %s - %s - Body: %s", resp.StatusCode, u.String(), err, string(ball))
	}

	return bytes.NewReader(ball), nil
}
