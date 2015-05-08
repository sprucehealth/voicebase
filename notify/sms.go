package notify

import (
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

func (n *NotificationManager) sendSMS(toNumber, message string) error {
	if n.smsAPI == nil {
		return nil
	}

	dispatch.RunAsync(func() {
		if err := n.smsAPI.Send(n.fromNumber, toNumber, message); err != nil {
			n.statSMSFailed.Inc(1)
			golog.Errorf("Error sending sms for message '%s': %s", message, err.Error())
		} else {
			n.statSMSSent.Inc(1)
		}
	})

	return nil
}
