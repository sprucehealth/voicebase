package notify

func (n *NotificationManager) sendSMSToUser(toNumber, message string) error {
	if n.twilioClient == nil {
		return nil
	}

	_, _, err := n.twilioClient.Messages.SendSMS(n.fromNumber, toNumber, message)
	if err != nil {
		n.statSMSFailed.Inc(1)
	} else {
		n.statSMSSent.Inc(1)
	}
	return err
}
