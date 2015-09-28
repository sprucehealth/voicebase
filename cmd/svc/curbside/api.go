package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
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

func writeAPIError(w http.ResponseWriter, message string, errorType string, code int) {
	json, err := json.Marshal(&apiErrorContainer{
		Err: apiError{
			Message: message,
			Type:    errorType,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, string(json), code)
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var d joinCommunityPOSTRequest
	var err error
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&d)
	if err != nil {
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

	err = postToSlack("Sloctorbot", message, c.SlackWebhookURL)
	if err != nil {
		writeAPIError(w, "We were unable to process your information. Please double check everything and try again.", "form_validation", http.StatusBadRequest)
		return
	}

	var data []byte
	data, _ = json.Marshal(&struct{ result string }{result: "success"})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
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

func postToSlack(username string, message string, url string) (err error) {
	if url == "" {
		return errors.New("url must not be blank when posting to Slack")
	}
	data, err := json.Marshal(&struct {
		Text      string `json:"text"`
		Username  string `json:"username"`
		IconEmoji string `json:"icon_emoji"`
	}{
		Text:      message,
		Username:  username,
		IconEmoji: ":orly:",
	})
	if err != nil {
		return err
	}
	res, err := http.Post(url, "text/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			b = nil
		}
		return fmt.Errorf("Bad status code %d from Slack: %s", res.StatusCode, string(b))
	}

	return nil
}
