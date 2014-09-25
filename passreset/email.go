package passreset

import (
	"fmt"
	"net/mail"
	"net/url"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
)

const (
	requestEmailKey = "passreset-request"
	successEmailKey = "passreset-success"
)

func init() {
	email.MustRegisterType(&email.Type{
		Key:  requestEmailKey,
		Name: "Password Reset Request",
		TestContext: &requestEmailContext{
			ResetURL: "https://www.sprucehealth.com/reset-password/verify",
		},
	})
	email.MustRegisterType(&email.Type{
		Key:  successEmailKey,
		Name: "Password Reset Success",
		TestContext: &successEmailContext{
			SupportEmail: "support@sprucehealth.com",
		},
	})
}

type requestEmailContext struct {
	ResetURL string
}

type successEmailContext struct {
	SupportEmail string
}

const (
	lostPasswordExpires     = 30 * 60 // seconds
	lostPasswordCodeExpires = 10 * 60 // seconds
	resetPasswordExpires    = 10 * 60 // seconds
)

func SendPasswordResetEmail(authAPI api.AuthAPI, emailService email.Service, domain string, accountID int64, emailAddress, supportEmail string) error {
	// Generate a temporary token that allows access to the password reset page
	token, err := authAPI.CreateTempToken(accountID, lostPasswordExpires, api.LostPassword, "")
	if err != nil {
		return err
	}

	params := url.Values{
		"token": []string{token},
		"email": []string{emailAddress},
	}
	return emailService.SendTemplateType(&mail.Address{Address: emailAddress}, requestEmailKey, &requestEmailContext{
		ResetURL: fmt.Sprintf("https://%s/reset-password/verify?%s", domain, params.Encode()),
	})
}

func SendPasswordHasBeenResetEmail(emailService email.Service, emailAddress, supportEmail string) error {
	return emailService.SendTemplateType(&mail.Address{Address: emailAddress}, successEmailKey, &successEmailContext{
		SupportEmail: supportEmail,
	})
}
