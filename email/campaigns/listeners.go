package campaigns

import (
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/patient"
)

var welcomeEmailEnabledDef = &cfg.ValueDef{
	Name:        "Email.Campaign.Welcome.Enabled",
	Description: "Enable or disable the welcome email.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

const patientSignupEmailType = "welcome"

func InitListeners(dispatch *dispatch.Dispatcher, cfgStore cfg.Store, emailService email.Service) {
	cfgStore.Register(welcomeEmailEnabledDef)
	dispatch.SubscribeAsync(func(ev *patient.SignupEvent) error {
		if cfgStore.Snapshot().Bool(welcomeEmailEnabledDef.Name) {
			if _, err := emailService.Send([]int64{ev.AccountID}, patientSignupEmailType, nil, &mandrill.Message{}, email.OnlyOnce|email.CanOptOut); err != nil {
				golog.Errorf("Failed to sent welcome email to account %d: %s", ev.AccountID, err)
			}
		}
		return nil
	})
}
