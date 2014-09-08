package notify

import (
	"net/mail"

	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	unsuitableEmailType = "unsuitable-for-spruce"
)

type unsuitableEmailTypeContext struct {
	PatientVisitID int64
}

func init() {
	email.MustRegisterType(&email.Type{
		Key:  unsuitableEmailType,
		Name: "Unsuitable for Spruce",
		TestContext: &unsuitableEmailTypeContext{
			PatientVisitID: 1,
		},
	})
}

func (n *NotificationManager) SendEmail(to *mail.Address, typeKey string, ctx interface{}) error {
	go func() {
		if err := n.emailService.SendTemplateType(to, typeKey, ctx); err != nil {
			golog.Errorf(err.Error())
			n.statEmailFailed.Inc(1)
		} else {
			n.statEmailSent.Inc(1)
		}
	}()
	return nil
}
