package support

import (
	"carefront/common/config"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/notify"
	"carefront/patient_visit"
)

func InitListeners(technicalSupportEmail, customerSupportEmail string, notificationManager *notify.NotificationManager) {
	dispatch.Default.Subscribe(func(ev *config.PanicEvent) error {
		if err := notificationManager.NotifySupport(technicalSupportEmail, ev); err != nil {
			golog.Errorf("Unable to notify support of a panic event: " + err.Error())
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		if err := notificationManager.NotifySupport(customerSupportEmail, ev); err != nil {
			golog.Errorf("Unable to notify support of a unsuitable visit: " + err.Error())
			return err
		}
		return nil
	})
}
