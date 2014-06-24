package doctor_queue

import (
	"carefront/api"
	"carefront/common"
	"carefront/patient_visit"
)

func routeIncomingPatientVisit(ev *patient_visit.VisitSubmittedEvent, dataAPI api.DataAPI) error {
	// get the patient case that the visit belongs to
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(ev.VisitId)
	if err != nil {
		return err
	}

	if patientCase.Status == common.PCStatusUnclaimed {
		// insert item into the unclaimed item queue given that it has not been claimed by a doctor yet
		patient, err := dataAPI.GetPatientFromId(ev.PatientId)
		if err != nil {
			return err
		}

		careProvidingStateId, err := dataAPI.GetCareProvidingStateId(patient.StateFromZipCode, patientCase.HealthConditionId.Int64())
		if err != nil {
			return err
		}

		if err := dataAPI.InsertUnclaimedItemIntoQueue(&api.DoctorQueueItem{
			CareProvidingStateId: careProvidingStateId,
			ItemId:               ev.VisitId,
			EventType:            api.DQEventTypePatientVisit,
			Status:               api.STATUS_PENDING,
			PatientCaseId:        patientCase.Id.Int64(),
		}); err != nil {
			return err
		}
	} else {
		doctorAssignments, err := dataAPI.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
		if err != nil {
			return err
		}
		// route it to the active doctor under the case
		for _, assignment := range doctorAssignments {
			if assignment.Status == api.STATUS_ACTIVE {
				if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
					DoctorId:  assignment.ProviderId,
					ItemId:    ev.VisitId,
					Status:    api.STATUS_PENDING,
					EventType: api.DQEventTypePatientVisit,
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
