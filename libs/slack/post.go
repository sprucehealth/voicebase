package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sprucehealth/backend/libs/errors"
)

// Message is posted to an incoming webhook
type Message struct {
	// Text is the main content of the message. To include links can format them
	// as `<https://example.com|Title of Link>` with the title being optional.
	Text string `json:"text"`
	// the following are optional
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// Post sends a message to a Slack webhook URL
func Post(webhookURL string, msg *Message) error {
	if webhookURL == "" {
		return errors.New("slack.Post: webhookURL must not be blank")
	}
	if msg == nil || msg.Text == "" {
		return errors.New("slack.Post: msg nil or msg text empty")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return errors.Trace(err)
	}
	res, err := http.Post(webhookURL, "appication/json", bytes.NewReader(data))
	if err != nil {
		return errors.Trace(err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("slack.Post: bad status code %d from Slack: %s", res.StatusCode, string(b))
	}
	return nil
}
