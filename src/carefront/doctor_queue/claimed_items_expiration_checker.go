package doctor_queue

import (
	"carefront/api"
	"carefront/libs/golog"
	"math/rand"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

var (
	// ExpireDuration is the maximum time between actions on the patient case that the doctor
	// has to maintain their claim on the case.
	ExpireDuration = 15 * time.Minute

	// GracePeriod is to ensure that any pending/ongoing requests
	// have ample time to complete before yanking access from
	// doctors who's claim on the case has expired
	GracePeriod = 5 * time.Minute

	// timePeriodBetweenChecks is the frequency with which the checker runs
	timePeriodBetweenChecks = 5 * time.Minute
)

// StartClaimedItemsForExpirationChecker runs periodically to revoke access
// to any temporarily claimed cases where the doctor has remained inactive for
// an extended period of time. In such a sitution, the exclusive access to the case
// is revoked and the item is placed back into the global queue for any elligible doctor to claim
func StartClaimedItemsExpirationChecker(dataAPI api.DataAPI, statsRegistry metrics.Registry) {
	go func() {

		claimExpirationSuccess := metrics.NewCounter()
		claimExpirationFailure := metrics.NewCounter()
		statsRegistry.Add("claim_expiration/failure", claimExpirationFailure)
		statsRegistry.Add("claim_expiration/success", claimExpirationSuccess)

		for {
			CheckForExpiredClaimedItems(dataAPI, claimExpirationSuccess, claimExpirationFailure)

			// add a random number of seconds to the time period to further reduce the probability that the
			// workers run on different systems in the same second, thereby introducing potential collision
			time.Sleep(timePeriodBetweenChecks + (time.Duration(rand.Intn(30)) * time.Second))
		}
	}()
}

func CheckForExpiredClaimedItems(dataAPI api.DataAPI, claimExpirationSuccess, claimExpirationFailure metrics.Counter) {
	// get currently claimed items in global queue
	claimedItems, err := dataAPI.GetClaimedItemsInQueue()
	if err != nil {
		golog.Errorf("Unable to get claimed items from global queue")
	}

	// iterate through items to check if any of the claims have expired
	for _, item := range claimedItems {
		if item.Expires.Add(GracePeriod).Before(time.Now()) {
			if err := revokeAccesstoCaseFromDoctor(item.PatientCaseId, item.DoctorId, dataAPI); err != nil {
				claimExpirationFailure.Inc(1)
				golog.Errorf("Unable to revoke access of case from doctor: %s", err)
			}
			claimExpirationSuccess.Inc(1)
		}
	}
}

func revokeAccesstoCaseFromDoctor(patientCaseId, doctorId int64, dataAPI api.DataAPI) error {
	if err := dataAPI.RevokeDoctorAccessToCase(patientCaseId, doctorId); err != nil {
		return err
	}

	// delete any treatment plan drafts that the doctor may have created
	if err := dataAPI.DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId); err != nil {
		return err
	}

	return nil
}
