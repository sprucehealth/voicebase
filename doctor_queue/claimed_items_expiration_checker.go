package doctor_queue

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

// StartClaimedItemsForExpirationChecker runs periodically to revoke access
// to any temporarily claimed cases where the doctor has remained inactive for
// an extended period of time. In such a sitution, the exclusive access to the case
// is revoked and the item is placed back into the global queue for any elligible doctor to claim
func StartClaimedItemsExpirationChecker(dataAPI api.DataAPI, analyticsLogger analytics.Logger, statsRegistry metrics.Registry) {
	go func() {
		claimExpirationSuccess := metrics.NewCounter()
		claimExpirationFailure := metrics.NewCounter()
		statsRegistry.Add("claim_expiration/failure", claimExpirationFailure)
		statsRegistry.Add("claim_expiration/success", claimExpirationSuccess)

		for {
			CheckForExpiredClaimedItems(dataAPI, analyticsLogger, claimExpirationSuccess, claimExpirationFailure)

			// add a random number of seconds to the time period to further reduce the probability that the
			// workers run on different systems in the same second, thereby introducing potential collision
			time.Sleep(timePeriodBetweenChecks + (time.Duration(rand.Intn(30)) * time.Second))
		}
	}()
}

func CheckForExpiredClaimedItems(dataAPI api.DataAPI, analyticsLogger analytics.Logger, claimExpirationSuccess, claimExpirationFailure *metrics.Counter) {
	// get currently claimed items in global queue
	claimedItems, err := dataAPI.GetClaimedItemsInQueue()
	if err != nil {
		golog.Errorf("Unable to get claimed items from global queue")
		return
	}

	// iterate through items to check if any of the claims have expired
	for _, item := range claimedItems {
		if item.Expires.Add(GracePeriod).Before(time.Now()) {
			patientCase, err := dataAPI.GetPatientCaseFromId(item.PatientCaseId)
			if err != nil {
				claimExpirationFailure.Inc(1)
				golog.Errorf("Unable to get patient case from id :%s", err)
				return
			}

			if err := revokeAccesstoCaseFromDoctor(item.PatientCaseId, patientCase.PatientId.Int64(), item.DoctorId, dataAPI); err != nil {
				claimExpirationFailure.Inc(1)
				golog.Errorf("Unable to revoke access of case from doctor: %s", err)
				return
			}

			jsonData, _ := json.Marshal(map[string]interface{}{
				"expiration_time": item.Expires,
			})

			analyticsLogger.WriteEvents([]analytics.Event{
				&analytics.ServerEvent{
					Event:     "jbcq_claim_revoke",
					Timestamp: analytics.Time(time.Now()),
					DoctorID:  item.DoctorId,
					CaseID:    patientCase.Id.Int64(),
					ExtraJSON: string(jsonData),
				},
			})
			claimExpirationSuccess.Inc(1)
		}
	}
}

func revokeAccesstoCaseFromDoctor(patientCaseId, patientId, doctorId int64, dataAPI api.DataAPI) error {
	if err := dataAPI.RevokeDoctorAccessToCase(patientCaseId, patientId, doctorId); err != nil {
		return err
	}

	// delete any treatment plan drafts that the doctor may have created
	if err := dataAPI.DeleteDraftTreatmentPlanByDoctorForCase(doctorId, patientCaseId); err != nil {
		return err
	}

	return nil
}
