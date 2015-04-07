package auth

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

var (
	ErrNoCellPhone = errors.New("no cellphone number for account")
)

func GenerateSMSCode() (string, error) {
	return common.GenerateRandomNumber(999999, 6)
}

// SendTwoFactorCode generates and sends a code to the cellphone number attached to the account. It also
// creates a temporary token linked to the code that can be used to verify a future request given the code.
func SendTwoFactorCode(authAPI api.AuthAPI, smsAPI api.SMSAPI, fromNumber string, accountID int64, deviceID string, expiration int) (string, error) {
	numbers, err := authAPI.GetPhoneNumbersForAccount(accountID)
	if err != nil {
		return "", err
	}

	var toNumber string
	for _, n := range numbers {
		if n.Type == api.PhoneCell {
			toNumber = n.Phone.String()
			break
		}
	}
	if toNumber == "" {
		return "", ErrNoCellPhone
	}

	code, err := GenerateSMSCode()
	if err != nil {
		return "", err
	}

	if _, err := authAPI.CreateTempToken(accountID, expiration, api.TwoFactorAuthCode, TwoFactorCodeToken(accountID, deviceID, code)); err != nil {
		return "", err
	}

	if err := smsAPI.Send(fromNumber, toNumber, fmt.Sprintf("Your Spruce verification code is %s", code)); err != nil {
		return "", err
	}

	return toNumber, nil
}

func TwoFactorCodeToken(accountID int64, deviceID, code string) string {
	return fmt.Sprintf("%d:%s:%s", accountID, deviceID, code)
}
