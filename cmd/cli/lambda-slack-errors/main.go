package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/slack"
)

// slackWebhookURL is the URL of the webhook to post events to.
var slackWebhookURL string

func init() {
	flag.StringVar(&slackWebhookURL, "slack.webhookurl", slackWebhookURL, "Slack webhook `URL` to post events")
}

type event struct {
	Records []*record
}

type record struct {
	EventVersion         string
	EventSubscriptionArn string
	EventSource          string
	SNS                  struct {
		SignatureVersion  string
		Timestamp         time.Time
		Signature         string
		SigningCertURL    string `json:"SigningCertUrl"`
		MessageID         string `json:"MessageId"`
		Message           string
		MessageAttributes map[string]struct {
			Type  string
			Value string
		}
		Type           string
		UnsubscribeURL string `json:"UnsubscribeUrl"`
		TopicARN       string `json:"TopicArn"`
		Subject        string
	} `json:"Sns"`
}

type errorEvent struct {
	Time    string `json:"t"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Src     string `json:"src"`
}

func main() {
	log.SetFlags(log.Lshortfile)
	boot.ParseFlags("")

	if slackWebhookURL == "" {
		log.Fatal("slack webhook URL not provided")
	}

	var ev event
	if err := json.NewDecoder(os.Stdin).Decode(&ev); err != nil {
		log.Fatalf("Failed to decode event: %s", err)
	}
	for _, rec := range ev.Records {
		msg := rec.SNS.Message
		var ee errorEvent
		if err := json.Unmarshal([]byte(msg), &ee); err == nil && ee.Message != "" {
			msg = fmt.Sprintf("*[%s] %s* : %s\n```%s```", ee.Level, ee.Src, ee.Time, ee.Message)
		}
		if err := slack.Post(slackWebhookURL, &slack.Message{
			Text:      msg,
			Username:  "Robot B-9",
			IconEmoji: ":robot:",
		}); err != nil {
			log.Printf("Failed to post: %s", err)
		}
	}
}
