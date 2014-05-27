package notify

import (
	"github.com/samuel/go-metrics/metrics"
	"github.com/subosito/twilio"
)

func sendSMSToUser(twilioClient *twilio.Client, fromNumber, toNumber string, message string, statFailed, statSent metrics.Counter) error {
	_, _, err := twilioClient.Messages.SendSMS(fromNumber, toNumber, message)
	if err != nil {
		statFailed.Inc(1)
	} else {
		statSent.Inc(1)
	}
	return err
}
