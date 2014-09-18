package notify

import "github.com/sprucehealth/backend/libs/golog"

func (n *NotificationManager) sendSMSToUser(toNumber, message string) error {
	if n.smsAPI == nil {
		return nil
	}

	go func() {
		if err := n.smsAPI.Send(n.fromNumber, toNumber, message); err != nil {
			n.statSMSFailed.Inc(1)
			golog.Errorf("Error sending sms: %s", err.Error())
		} else {
			n.statSMSSent.Inc(1)
		}
	}()
	return nil
}
