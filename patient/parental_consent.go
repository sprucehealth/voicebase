package patient

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	parentalConsentTokenPurpose    = "ParentalConsent"
	parentalConsentTokenExpiration = time.Hour * 24 * 14
	txtParentalConsentRequestSMS   = "parental_consent_request_sms"
)

// ParentalConsentCompleted takes care of updating a child's patient account and visits
// once a parent has completed the consent flow.
func ParentalConsentCompleted(dataAPI api.DataAPI, publisher dispatch.Publisher, parentPatientID, childPatientID common.PatientID) error {
	newlyCompleted, err := dataAPI.ParentalConsentCompletedForPatient(childPatientID)
	if err != nil {
		return errors.Trace(err)
	}
	if newlyCompleted {
		publisher.PublishAsync(&ParentalConsentCompletedEvent{
			ParentPatientID: parentPatientID,
			ChildPatientID:  childPatientID,
		})
	}
	return nil
}

// ParentalConsentURL returns the URL for the parental consent web page
func ParentalConsentURL(tokensAPI api.Tokens, webDomain string, childPatientID common.PatientID) (string, error) {
	token, err := GenerateParentalConsentToken(tokensAPI, childPatientID)
	if err != nil {
		return "", errors.Trace(err)
	}
	params := url.Values{"t": []string{token}}
	return fmt.Sprintf("https://%s/pc/%d?%s", webDomain, childPatientID.Uint64(), params.Encode()), nil
}

// ParentalConsentRequestSMSAction returns the action URL for requesting consent from a parent for a child to get treatment.
func ParentalConsentRequestSMSAction(dataAPI api.DataAPI, webDomain string, childPatientID common.PatientID) (*app_url.SpruceAction, error) {
	consentURL, err := ParentalConsentURL(dataAPI, webDomain, childPatientID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	text, err := dataAPI.LocalizedText(api.LanguageIDEnglish, []string{txtParentalConsentRequestSMS})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return app_url.ComposeSMSAction(fmt.Sprintf(text[txtParentalConsentRequestSMS], consentURL)), nil
}

// GenerateParentalConsentToken returns a token that can be used to validate access to the parent/child consent flow
func GenerateParentalConsentToken(tokensAPI api.Tokens, childPatientID common.PatientID) (string, error) {
	token, err := tokensAPI.CreateToken(parentalConsentTokenPurpose, strconv.FormatUint(childPatientID.Uint64(), 10), "", parentalConsentTokenExpiration)
	return token, errors.Trace(err)
}

// ValidateParentalConsentToken returns true iff the token is valid for the child's patient ID
func ValidateParentalConsentToken(tokensAPI api.Tokens, token string, childPatientID common.PatientID) bool {
	idStr, err := tokensAPI.ValidateToken(parentalConsentTokenPurpose, token)
	if err != nil {
		if err != api.ErrTokenDoesNotExist && err != api.ErrTokenExpired {
			golog.Errorf("Error validating parental consent token: %s", err)
		}
		return false
	}
	id, _ := strconv.ParseUint(idStr, 10, 64)
	return id != 0 && id == childPatientID.Uint64()
}
