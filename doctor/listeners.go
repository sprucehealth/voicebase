package doctor

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/patient_visit"
)

func InitListeners(dataAPI api.DataAPI, apiDomain string, dispatcher *dispatch.Dispatcher) {
	dispatcher.SubscribeAsync(func(ev *doctor_treatment_plan.TreatmentPlanSubmittedEvent) error {
		// check for any submitted/treated visits for the case
		states := common.SubmittedPatientVisitStates()
		states = append(states, common.TreatedPatientVisitStates()...)
		visits, err := dataAPI.GetVisitsForCase(ev.TreatmentPlan.PatientCaseID.Int64(), states)
		if err != nil {
			return err
		}

		// ensure that a single doctor transaction exists for every submitted visit.
		for _, visit := range visits {

			_, err := dataAPI.TransactionForItem(visit.ID.Int64(), ev.TreatmentPlan.DoctorID.Int64(), visit.SKUType)
			if !api.IsErrNotFound(err) && err != nil {
				return err
			} else if err == nil {
				continue
			}

			if err := createDoctorTransaction(dataAPI, ev.TreatmentPlan.DoctorID.Int64(),
				ev.TreatmentPlan.PatientID, visit); err != nil {
				return err
			}
		}
		return nil
	})

	dispatcher.SubscribeAsync(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		visit, err := dataAPI.GetPatientVisitFromID(ev.PatientVisitID)
		if err != nil {
			return err
		}

		if err := createDoctorTransaction(dataAPI, ev.DoctorID, ev.PatientID, visit); err != nil {
			return err
		}
		return nil
	})

	dispatcher.SubscribeAsync(func(ev *DoctorLoggedInEvent) error {
		if err := promotions.CreateReferralProgramForDoctor(ev.Doctor, dataAPI, apiDomain); err != nil {
			return err
		}
		return nil
	})

}

func createDoctorTransaction(dataAPI api.DataAPI, doctorID, patientID int64, visit *common.PatientVisit) error {

	var itemCostID *int64
	// lookup the patient receipt to get the itemCostID associated with the
	// visit. If one doesn't exist, then treat it as no cost existing for the visit
	patientReceipt, err := dataAPI.GetPatientReceipt(patientID, visit.ID.Int64(), visit.SKUType, false)
	if err == nil {
		itemCostID = &patientReceipt.ItemCostID
	} else if err != nil && !api.IsErrNotFound(err) {
		return err
	}

	if err := dataAPI.CreateDoctorTransaction(&common.DoctorTransaction{
		DoctorID:   doctorID,
		ItemCostID: itemCostID,
		SKUType:    visit.SKUType,
		ItemID:     visit.ID.Int64(),
		PatientID:  patientID,
	}); err != nil {
		return err
	}

	return nil
}
