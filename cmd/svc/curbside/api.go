package main

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/slack"
)

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type apiErrorContainer struct {
	Err apiError `json:"error"`
}

func init() {
	flag.StringVar(&c.SlackWebhookURL, "slack_webhook_url", "", "Slack Incoming Webhook URL")
}

func writeAPIError(w http.ResponseWriter, message, errorType string, code int) {
	httputil.JSONResponse(w, code, &apiErrorContainer{
		Err: apiError{
			Message: message,
			Type:    errorType,
		},
	})
}

func submitHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var d joinCommunityPOSTRequest
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeAPIError(w, "We were unable to process your information. Please double check everything and try again.", "form_validation", http.StatusBadRequest)
		return
	}

	envWarning := ""
	if c.Environment != "prod" {
		envWarning = " (Environment: " + c.Environment + ")"
	}

	textStrings := []string{
		"*New Curbside Application" + envWarning + "*\n\n",
		"_First Name:_\n" + d.FirstName,
		"_Last Name:_\n" + d.LastName,
		"_Email:_\n" + d.Email,
		"_Licensed Locations:_\n" + d.LicensedLocations,
		"_Reasons Interested:_\n" + d.ReasonsInterested,
		"_Dermatology Interested:_\n" + d.DermatologyInterests,
		"_Referral Source:_\n" + d.ReferralSource,
	}
	message := strings.Join(textStrings, "\n\n")

	if err := slack.Post(c.SlackWebhookURL, &slack.Message{Text: message, Username: "Sloctorbot", IconEmoji: ":orly:"}); err != nil {
		writeAPIError(w, "We were unable to process your information. Please double check everything and try again.", "form_validation", http.StatusBadRequest)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct {
		Result string `json:"result"`
	}{
		Result: "success",
	})
}

func (d *joinCommunityPOSTRequest) Validate() error {
	if d.FirstName == "" {
		return errors.New("Please enter your first name.")
	}
	if d.LastName == "" {
		return errors.New("Please enter your last name.")
	}
	if d.Email == "" {
		return errors.New("Please enter your email address.")
	}
	if d.LicensedLocations == "" {
		return errors.New("Please enter where you are licensed.")
	}
	if d.ReasonsInterested == "" {
		return errors.New("Please enter the reason you are interested in joining.")
	}
	if d.DermatologyInterests == "" {
		return errors.New("Please enter your interest areas within dermatology.")
	}
	if d.ReferralSource == "" {
		return errors.New("Please tell us how you heard about us.")
	}

	return nil
}
