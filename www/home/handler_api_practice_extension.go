package home

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type demoRequestAPIHandler struct {
	cfg cfg.Store
}

type whitepaperRequestAPIHandler struct {
	cfg cfg.Store
}

type demoPOSTRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	State     string `json:"state"`
}

type whitepaperPOSTRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

var whitepaperRequestSlackWebhookURLDef = &cfg.ValueDef{
	Name:        "SlackURL.Webhook.PracticeExtension.WhitepaperRequest",
	Description: "A Slack webhook URL to post the details of a person requesting the Practice Extension whitepaper.",
	Type:        cfg.ValueTypeString,
	Default:     "",
}

var demoRequestSlackWebhookURLDef = &cfg.ValueDef{
	Name:        "SlackURL.Webhook.PracticeExtension.DemoRequest",
	Description: "A Slack webhook URL to post the details of a person requesting a demo of Practice Extension.",
	Type:        cfg.ValueTypeString,
	Default:     "",
}

func init() {
	config.MustRegisterCfgDef(whitepaperRequestSlackWebhookURLDef)
	config.MustRegisterCfgDef(demoRequestSlackWebhookURLDef)
}

func (d *demoPOSTRequest) Validate() error {
	if d.FirstName == "" {
		return errors.New("Please enter your first name.")
	}
	if d.LastName == "" {
		return errors.New("Please enter your last name.")
	}
	if d.Email == "" {
		return errors.New("Please enter your email address.")
	}
	if d.Phone == "" {
		return errors.New("Please enter your phone number.")
	}
	if d.State == "" {
		return errors.New("Please enter where you are licensed.")
	}
	return nil
}

func (d *whitepaperPOSTRequest) Validate() error {
	if d.FirstName == "" {
		return errors.New("Please enter your first name.")
	}
	if d.LastName == "" {
		return errors.New("Please enter your last name.")
	}
	if d.Email == "" {
		return errors.New("Please enter your email address.")
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

func newPracticeExtensionDemoAPIHandler(cfg cfg.Store) httputil.ContextHandler {
	return httputil.SupportedMethods(&demoRequestAPIHandler{cfg: cfg}, httputil.Post)
}

func newPracticeExtensionWhitepaperAPIHandler(cfg cfg.Store) httputil.ContextHandler {
	return httputil.SupportedMethods(&whitepaperRequestAPIHandler{cfg: cfg}, httputil.Post)
}

func (h *demoRequestAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var d demoPOSTRequest
	var err error
	if err = json.NewDecoder(r.Body).Decode(&d); err != nil {
		golog.Errorf("Error parsing Practice Extension Demo Request: %s", err.Error())
		www.APIBadRequestError(w, r, "We were unable to process your information. Please double check everything and try again.")
		return
	}
	err = d.Validate()
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	textStrings := []string{
		"*New Practice Extension Demo Request*\n\n",
		"_First Name:_\n" + d.FirstName,
		"_Last Name:_\n" + d.LastName,
		"_Email:_\n" + d.Email,
		"_Phone:_\n" + d.Phone,
		"_State:_\n" + d.State,
	}
	text := strings.Join(textStrings, "\n\n")

	url := h.cfg.Snapshot().String(demoRequestSlackWebhookURLDef.Name)
	if err := postToSlack("DERPebot", text, url); err != nil {
		golog.Errorf("Failed to post whitepaper request form data to Slack; however, we did not return a 500. Error: %s", err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

func (h *whitepaperRequestAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var d whitepaperPOSTRequest
	var err error
	if err = json.NewDecoder(r.Body).Decode(&d); err != nil {
		golog.Errorf("Error parsing Practice ExtensionWhitepaper Request: %s", err.Error())
		www.APIBadRequestError(w, r, "We were unable to process your information. Please double check everything and try again.")
		return
	}
	err = d.Validate()
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}
	err = d.Validate()

	textStrings := []string{
		"*New Practice Extension Whitepaper Download*\n\n",
		"_First Name:_\n" + d.FirstName,
		"_Last Name:_\n" + d.LastName,
		"_Email:_\n" + d.Email,
	}
	text := strings.Join(textStrings, "\n\n")

	go func() {
		url := h.cfg.Snapshot().String(whitepaperRequestSlackWebhookURLDef.Name)
		if err = postToSlack("DERPebot", text, url); err != nil {
			// We silently fail because we don't want to let Slack errors block users from downloading the whitepaper
			golog.Errorf("Failed to post whitepaper request form data to Slack; however, we did not return a 500. Error: %s", err)
		}
	}()

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
