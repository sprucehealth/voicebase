package campaigns

import (
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/patient"
)

const patientSignupEmailType = "welcome"

func InitListeners(dispatch *dispatch.Dispatcher, emailService email.Service) {
	dispatch.SubscribeAsync(func(ev *patient.SignupEvent) error {
		if _, err := emailService.Send([]int64{ev.AccountID}, patientSignupEmailType, nil, &mandrill.Message{}, email.OnlyOnce|email.CanOptOut); err != nil {
			golog.Errorf("Failed to sent welcome email to account %d: %s", ev.AccountID, err)
		}
		return nil
	})
}
