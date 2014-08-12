package support

import (
	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/notify"
)

func InitListeners(technicalSupportEmail, customerSupportEmail string, notificationManager *notify.NotificationManager) {
	dispatch.Default.Subscribe(func(ev *config.PanicEvent) error {
		if err := notificationManager.NotifySupport(technicalSupportEmail, ev); err != nil {
			golog.Errorf("Unable to notify support of a panic event: " + err.Error())
			return err
		}
		return nil
	})
}
