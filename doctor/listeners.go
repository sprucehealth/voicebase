package doctor

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/sku"
)

func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) {
	dispatcher.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		// being the first treatment plan for the patient, this marks the completion of the doctor transaction
		if ev.TreatmentPlan.Parent.ParentType == common.TPParentTypePatientVisit {
			go func() {
				if err := createDoctorTransaction(dataAPI, ev.DoctorId, ev.PatientId, ev.TreatmentPlan.Parent.ParentId.Int64()); err != nil {
					golog.Errorf(err.Error())
					return
				}
			}()
		}
		return nil
	})

	dispatcher.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		go func() {
			if err := createDoctorTransaction(dataAPI, ev.DoctorID, ev.PatientID, ev.PatientVisitID); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}()
		return nil
	})

	dispatcher.Subscribe(func(ev *DoctorLoggedInEvent) error {
		go func() {
			if err := promotions.CreateReferralProgramForDoctor(ev.Doctor, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}()
		return nil
	})

}

func createDoctorTransaction(dataAPI api.DataAPI, doctorID, patientID, patientVisitID int64) error {

	var itemCostId *int64
	// lookup the patient receipt to get the itemCostID associated with the
	// visit. If one doesn't exist, then treat it as no cost existing for the visit
	patientReceipt, err := dataAPI.GetPatientReceipt(patientID, patientVisitID, sku.AcneVisit, false)
	if err == nil {
		itemCostId = &patientReceipt.ItemCostID
	} else if err != nil && err != api.NoRowsError {
		return err
	}

	if err := dataAPI.CreateDoctorTransaction(&common.DoctorTransaction{
		DoctorID:   doctorID,
		ItemCostID: itemCostId,
		ItemType:   sku.AcneVisit,
		ItemID:     patientVisitID,
		PatientID:  patientID,
	}); err != nil {
		return err
	}

	return nil
}
