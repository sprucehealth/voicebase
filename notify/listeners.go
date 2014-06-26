package notify

import (
	"errors"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/patient"
)

func InitListeners(dataAPI api.DataAPI) {
	// Notify the doctor when a patient submits a new visit
	dispatch.Default.Subscribe(func(ev *patient.AccountLoggedOutEvent) error {
		// delete any existing push notification communication preference
		// for a user that is logging out so that we are not sending push notifications to this device
		// when the user logs back in, we will re-register the device for push notifications
		if err := dataAPI.DeletePushCommunicationPreferenceForAccount(ev.AccountId); err != nil {
			return errors.New("Unable to delete communication preference for patient: " + err.Error())
		}
		return nil
	})
}
