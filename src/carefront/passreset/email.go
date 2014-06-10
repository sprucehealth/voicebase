package passreset

import (
	"carefront/api"
	"carefront/email"
	"fmt"
	"net/url"
)

const (
	lostPasswordExpires     = 30 * 60 // 30 min
	lostPasswordCodeExpires = 10 * 60 // 10 min
	resetPasswordExpires    = 10 * 60 // 5 min
)

func SendPasswordResetEmail(authAPI api.AuthAPI, emailService email.Service, domain string, accountID int64, emailAddress, fromEmail string) error {
	// Generate a temporary token that allows access to the password reset page
	token, err := authAPI.CreateTempToken(accountID, lostPasswordExpires, api.LostPassword, "")
	if err != nil {
		return err
	}

	params := url.Values{
		"token": []string{token},
		"email": []string{emailAddress},
	}
	resetURL := fmt.Sprintf("https://www.%s/reset-password/verify?%s", domain, params.Encode())

	em := &email.Email{
		From:    fromEmail,
		To:      emailAddress,
		Subject: "Reset your Spruce password",
		BodyText: `Hello,

We've received a request to reset your password. To reset your password click the link below.

` + resetURL,
	}

	return emailService.SendEmail(em)
}

func SendPasswordHasBeenResetEmail(emailService email.Service, emailAddress, fromEmail string) error {
	em := &email.Email{
		From:    fromEmail,
		To:      emailAddress,
		Subject: "Reset your Spruce password",
		BodyText: `Hello,

You've successfully changed your account password.

Thank you,
The Spruce Team

-
Need help? Contact support@sprucehealth.com`,
	}
	return emailService.SendEmail(em)
}
