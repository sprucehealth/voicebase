package home

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/www"
)

type statesByAbbr []*common.State

func (s statesByAbbr) Len() int           { return len(s) }
func (s statesByAbbr) Less(a, b int) bool { return s[a].Abbreviation < s[b].Abbreviation }
func (s statesByAbbr) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

type statesByName []*common.State

func (s statesByName) Len() int           { return len(s) }
func (s statesByName) Less(a, b int) bool { return s[a].Name < s[b].Name }
func (s statesByName) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }

func newParentalConsentCookie(childPatientID common.PatientID, token string, r *http.Request) *http.Cookie {
	return www.NewCookie(fmt.Sprintf("ct_%d", childPatientID.Uint64()), token, r)
}

func parentalConsentCookie(childPatientID common.PatientID, r *http.Request) string {
	cookie, err := r.Cookie(fmt.Sprintf("ct_%d", childPatientID.Uint64()))
	if err != nil {
		return ""
	}
	return cookie.Value
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
