package passreset

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/mandrill"
)

const (
	requestEmailType = "passreset-request"
	successEmailType = "passreset-success"
)

const (
	lostPasswordExpires     = 30 * 60 // seconds
	lostPasswordCodeExpires = 10 * 60 // seconds
	resetPasswordExpires    = 10 * 60 // seconds
)

// SendPasswordResetEmail sends the password reset email for an account
func SendPasswordResetEmail(authAPI api.AuthAPI, emailService email.Service, webDomain string, accountID int64) error {
	// Generate a temporary token that allows access to the password reset page
	token, err := authAPI.CreateTempToken(accountID, lostPasswordExpires, api.LostPassword, "")
	if err != nil {
		return err
	}

	params := url.Values{
		"token": []string{token},
		"id":    []string{strconv.FormatInt(accountID, 10)},
	}
	_, err = emailService.Send([]int64{accountID}, requestEmailType, nil, &mandrill.Message{
		GlobalMergeVars: []mandrill.Var{
			{
				Name:    "ResetURL",
				Content: fmt.Sprintf("https://%s/reset-password/verify?%s", webDomain, params.Encode()),
			},
		},
	}, 0)
	return err
}

// SendPasswordHasBeenResetEmail sends the email for when a password is reset through the reset password flow
func SendPasswordHasBeenResetEmail(emailService email.Service, accountID int64) error {
	_, err := emailService.Send([]int64{accountID}, successEmailType, nil, &mandrill.Message{}, 0)
	return err
}
