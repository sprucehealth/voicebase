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

func SendPasswordResetEmail(authAPI api.AuthAPI, emailService email.Service, domain string, accountID int64) error {
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
				Content: fmt.Sprintf("https://%s/reset-password/verify?%s", domain, params.Encode()),
			},
		},
	}, 0)
	return err
}

func SendPasswordHasBeenResetEmail(emailService email.Service, accountID int64) error {
	_, err := emailService.Send([]int64{accountID}, successEmailType, nil, &mandrill.Message{}, 0)
	return err
}
