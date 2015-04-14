package notify

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
)

func (n *NotificationManager) SendEmail(toAccountID int64, typeName string, vars []mandrill.Var) error {
	go func() {
		if _, err := n.emailService.Send([]int64{toAccountID}, typeName, nil, &mandrill.Message{
			GlobalMergeVars: vars,
		}, 0); err != nil {
			golog.Errorf(err.Error())
			n.statEmailFailed.Inc(1)
		} else {
			n.statEmailSent.Inc(1)
		}
	}()
	return nil
}
