package main

import (
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient"
)

type snsNotification struct {
	Default string `json:"default"`
	HTTP    string `json:"http"`
}

func InitNotifyListener(disp *dispatch.Dispatcher, snsCli *sns.SNS, topic string) {
	note := &snsNotification{Default: "VisitSubmitted", HTTP: "party/time"}
	disp.SubscribeAsync(func(ev *patient.VisitSubmittedEvent) error {
		if err := snsCli.Publish(note, topic); err != nil {
			golog.Warningf("SNS notification failed for party time: %s", err.Error())
		}
		return nil
	})
}
