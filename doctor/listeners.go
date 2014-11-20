package doctor

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) {
	dispatcher.SubscribeAsync(func(ev *doctor_treatment_plan.TreatmentPlanSubmittedEvent) error {
		// check for any submitted/treated visits for the case
		states := common.SubmittedPatientVisitStates()
		states = append(states, common.TreatedPatientVisitStates()...)
		visits, err := dataAPI.GetVisitsForCase(ev.TreatmentPlan.PatientCaseId.Int64(), states)
		if err != nil {
			return err
		}

		// ensure that a single doctor transaction exists for every submitted visit.
		for _, visit := range visits {

			_, err := dataAPI.TransactionForItem(visit.PatientVisitId.Int64(), ev.TreatmentPlan.DoctorId.Int64(), visit.SKU)
			if err != api.NoRowsError && err != nil {
				return err
			} else if err == nil {
				continue
			}

			if err := createDoctorTransaction(dataAPI, ev.TreatmentPlan.DoctorId.Int64(),
				ev.TreatmentPlan.PatientId, visit); err != nil {
				return err
			}
		}
		return nil
	})

	dispatcher.SubscribeAsync(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		visit, err := dataAPI.GetPatientVisitFromId(ev.PatientVisitID)
		if err != nil {
			return err
		}

		if err := createDoctorTransaction(dataAPI, ev.DoctorID, ev.PatientID, visit); err != nil {
			return err
		}
		return nil
	})

	dispatcher.SubscribeAsync(func(ev *DoctorLoggedInEvent) error {
		if err := promotions.CreateReferralProgramForDoctor(ev.Doctor, dataAPI); err != nil {
			return err
		}
		return nil
	})

}

func createDoctorTransaction(dataAPI api.DataAPI, doctorID, patientID int64, visit *common.PatientVisit) error {

	var itemCostId *int64
	// lookup the patient receipt to get the itemCostID associated with the
	// visit. If one doesn't exist, then treat it as no cost existing for the visit
	patientReceipt, err := dataAPI.GetPatientReceipt(patientID, visit.PatientVisitId.Int64(), visit.SKU, false)
	if err == nil {
		itemCostId = &patientReceipt.ItemCostID
	} else if err != nil && err != api.NoRowsError {
		return err
	}

	if err := dataAPI.CreateDoctorTransaction(&common.DoctorTransaction{
		DoctorID:   doctorID,
		ItemCostID: itemCostId,
		ItemType:   visit.SKU,
		ItemID:     visit.PatientVisitId.Int64(),
		PatientID:  patientID,
	}); err != nil {
		return err
	}

	return nil
}
