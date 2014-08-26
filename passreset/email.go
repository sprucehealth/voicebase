package passreset

import (
	"fmt"
	"net/url"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
)

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
	resetURL := fmt.Sprintf("https://%s/reset-password/verify?%s", domain, params.Encode())

	em := &email.Email{
		From:    supportEmail,
		To:      []string{emailAddress},
		Subject: "Reset your Spruce password",
		Text: []byte(`Hello,

We've received a request to reset your password. To reset your password click the link below.

` + resetURL),
	}

	return emailService.Send(em)
}

func SendPasswordHasBeenResetEmail(emailService email.Service, emailAddress, supportEmail string) error {
	em := &email.Email{
		From:    supportEmail,
		To:      []string{emailAddress},
		Subject: "Reset your Spruce password",
		Text: []byte(fmt.Sprintf(`Hello,

You've successfully changed your account password.

Thank you,
The Spruce Team

-
Need help? Contact %s`, supportEmail)),
	}
	return emailService.Send(em)
}
