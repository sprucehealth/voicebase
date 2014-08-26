package notify

import "github.com/sprucehealth/backend/email"

func (n *NotificationManager) SendEmail(em *email.Email) error {
	go func() {
		if err := n.emailService.Send(em); err != nil {
			n.statEmailFailed.Inc(1)
		} else {
			n.statEmailSent.Inc(1)
		}
	}()
	return nil
}
