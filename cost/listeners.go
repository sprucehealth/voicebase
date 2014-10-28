package cost

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) {

	dispatcher.Subscribe(func(ev *VisitChargedEvent) error {

		go func() {
			// looking up any existing referral tracking entry for this patient
			referralTrackingEntry, err := dataAPI.PendingReferralTrackingForAccount(ev.AccountID)
			if err == api.NoRowsError {
				// nothing to do here since there is no feedback to give
				return
			} else if err != nil {
				golog.Errorf(err.Error())
				return
			}

			// lookup the referral program
			referralProgram, err := dataAPI.ReferralProgram(referralTrackingEntry.CodeID, promotions.Types)
			if err != nil {
				golog.Errorf(err.Error())
				return
			}

			// update the referral program to indicate that the referred patient
			// submitted a visit
			if err := referralProgram.Data.(promotions.ReferralProgram).
				ReferredAccountSubmittedVisit(ev.AccountID, referralTrackingEntry.CodeID, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}()

		return nil
	})
}
