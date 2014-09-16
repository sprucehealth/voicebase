package doctor

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	twoFactorCodeDigits       = 6
	twoFactorCodeMax          = 999999
	twoFactorAuthTokenPurpose = "2FA"
	twoFactorAuthCodePurpose  = "2FACode"
)

type twoFactorHandler struct {
	authAPI             api.AuthAPI
	dataAPI             api.DataAPI
	smsAPI              api.SMSAPI
	fromNumber          string
	twoFactorExpiration int
}

type TwoFactorRequest struct {
	Token  string `json:"token"`
	Code   string `json:"code"`
	Resend bool   `json:"bool"`
}

func NewTwoFactorHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, smsAPI api.SMSAPI, fromNumber string, twoFactorExpiration int) http.Handler {
	return &twoFactorHandler{
		dataAPI:             dataAPI,
		authAPI:             authAPI,
		smsAPI:              smsAPI,
		fromNumber:          fromNumber,
		twoFactorExpiration: twoFactorExpiration,
	}
}

func (d *twoFactorHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	var req TwoFactorRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	account, err := d.authAPI.ValidateTempToken(twoFactorAuthTokenPurpose, req.Token)
	if err == api.TokenDoesNotExist || err == api.TokenExpired {
		return false, apiservice.NewAccessForbiddenError()
	} else if err != nil {
		return false, err
	}
	context := apiservice.GetContext(r)
	context.RequestCache[apiservice.Account] = account
	context.RequestCache[apiservice.RequestData] = &req
	return true, nil
}

func (d *twoFactorHandler) NonAuthenticated() bool {
	return true
}

func (d *twoFactorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := apiservice.GetContext(r)
	account := context.RequestCache[apiservice.Account].(*common.Account)
	req := context.RequestCache[apiservice.RequestData].(*TwoFactorRequest)

	appHeaders := apiservice.ExtractSpruceHeaders(r)

	if req.Resend {
		if _, err := sendTwoFactorCode(d.authAPI, d.smsAPI, d.fromNumber, account.ID, appHeaders.DeviceID, d.twoFactorExpiration); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
		return
	}

	account, err := d.authAPI.ValidateTempToken(twoFactorAuthCodePurpose, twoFactorCodeToken(account.ID, appHeaders.DeviceID, req.Code))
	if err == api.TokenDoesNotExist || err == api.TokenExpired {
		apiservice.WriteUserError(w, http.StatusForbidden, "Invalid verification code")
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Mark this device as being verified with two factor
	if err := d.authAPI.UpdateAccountDeviceVerification(account.ID, appHeaders.DeviceID, true); err != nil {
		// Don't return an error since the person can still continue even if this fails for whatever reason
		golog.Errorf(err.Error())
	}

	token, err := d.authAPI.CreateToken(account.ID, api.Mobile, api.RegularAuth)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountId(account.ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &AuthenticationResponse{Token: token, Doctor: doctor})
}

// sendTwoFactorCode generates and sends a code to the cellphone number attached to the account. It also
// creates a temporary token linked to the code that can be used to verify a future request given the code.
func sendTwoFactorCode(authAPI api.AuthAPI, smsAPI api.SMSAPI, fromNumber string, accountID int64, deviceID string, expiration int) (string, error) {
	numbers, err := authAPI.GetPhoneNumbersForAccount(accountID)
	if err != nil {
		return "", err
	}

	var toNumber string
	for _, n := range numbers {
		if n.Type == api.PHONE_CELL {
			toNumber = n.Phone.String()
			break
		}
	}
	if toNumber == "" {
		return "", errors.New("no cellphone number for account")
	}

	code, err := common.GenerateRandomNumber(twoFactorCodeMax, twoFactorCodeDigits)
	if err != nil {
		return "", err
	}

	if _, err := authAPI.CreateTempToken(accountID, expiration, twoFactorAuthCodePurpose, twoFactorCodeToken(accountID, deviceID, code)); err != nil {
		return "", err
	}

	if err := smsAPI.Send(fromNumber, toNumber, fmt.Sprintf("Your Spruce verification code is %s", code)); err != nil {
		return "", err
	}

	return toNumber, nil
}

func twoFactorCodeToken(accountID int64, deviceID, code string) string {
	return fmt.Sprintf("%d:%s:%s", accountID, deviceID, code)
}
