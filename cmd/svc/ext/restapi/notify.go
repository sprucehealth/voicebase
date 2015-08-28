package main

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient"
)

type snsNotification struct {
	Default string `json:"default"`
	HTTP    string `json:"http"`
}

func initNotifyListener(disp *dispatch.Dispatcher, snsCli *sns.SNS, topic string) {
	note := &snsNotification{Default: "VisitSubmitted", HTTP: "party/time"}
	noteJSONBytes, err := json.Marshal(note)
	if err != nil {
		panic(err)
	}
	noteJSON := string(noteJSONBytes)
	disp.SubscribeAsync(func(ev *patient.VisitSubmittedEvent) error {
		_, err := snsCli.Publish(&sns.PublishInput{
			Message:   &noteJSON,
			TargetArn: &topic,
		})
		if err != nil {
			golog.Warningf("SNS notification failed for party time: %s", err.Error())
		}
		return nil
	})
}
