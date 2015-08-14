package patient

import (
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type mockTokens_consent struct {
	api.Tokens
}

func (t *mockTokens_consent) CreateToken(purpose, key, token string, expires time.Duration) (string, error) {
	return purpose + key, nil
}

func (t *mockTokens_consent) ValidateToken(purpose, token string) (string, error) {
	if strings.HasPrefix(token, purpose) {
		return token[len(purpose):], nil
	}
	return "", api.ErrTokenDoesNotExist
}

func TestParentalConsentToken(t *testing.T) {
	tokens := &mockTokens_consent{}
	token, err := GenerateParentalConsentToken(tokens, common.NewPatientID(1))
	test.OK(t, err)
	test.Equals(t, "ParentalConsent1", token)
	test.Equals(t, true, ValidateParentalConsentToken(tokens, token, common.NewPatientID(1)))
	test.Equals(t, false, ValidateParentalConsentToken(tokens, token, common.NewPatientID(2)))
	test.Equals(t, false, ValidateParentalConsentToken(tokens, "abc", common.NewPatientID(1)))
	url, err := ParentalConsentURL(tokens, "domain", common.NewPatientID(3))
	test.OK(t, err)
	test.Equals(t, "https://domain/pc/3?t=ParentalConsent3", url)
}
