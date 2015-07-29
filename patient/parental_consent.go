package patient

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	parentalConsentTokenPurpose    = "ParentalConsent"
	parentalConsentTokenExpiration = time.Hour * 24 * 14
)

// ParentalConsentURL returns the URL for the parental consent web page
func ParentalConsentURL(tokensAPI api.Tokens, webDomain string, childPatientID int64) (string, error) {
	token, err := GenerateParentalConsentToken(tokensAPI, childPatientID)
	if err != nil {
		return "", err
	}
	params := url.Values{"t": []string{token}}
	return fmt.Sprintf("https://%s/pc/%d?%s", webDomain, childPatientID, params.Encode()), nil
}

// GenerateParentalConsentToken returns a token that can be used to validate access to the parent/child consent flow
func GenerateParentalConsentToken(tokensAPI api.Tokens, childPatientID int64) (string, error) {
	return tokensAPI.CreateToken(parentalConsentTokenPurpose, strconv.FormatInt(childPatientID, 10), "", parentalConsentTokenExpiration)
}

// ValidateParentalConsentToken returns true iff the token is valid for the child's patient ID
func ValidateParentalConsentToken(tokensAPI api.Tokens, token string, childPatientID int64) bool {
	idStr, err := tokensAPI.ValidateToken(parentalConsentTokenPurpose, token)
	if err != nil {
		if err != api.ErrTokenDoesNotExist && err != api.ErrTokenExpired {
			golog.Errorf("Error validating parental consent token: %s", err)
		}
		return false
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id != 0 && id == childPatientID
}
