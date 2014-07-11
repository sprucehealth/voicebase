package doctor_queue

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/patient_visit"
)

func routeIncomingPatientVisit(ev *patient_visit.VisitSubmittedEvent, dataAPI api.DataAPI) error {

	// get the patient's care team
	careTeam, err := dataAPI.GetCareTeamForPatient(ev.PatientId)
	if err != nil {
		golog.Errorf("Unable to get care team for patient: %s", err)
		return err
	}

	// get the patient case that the visit belongs to
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(ev.VisitId)
	if err != nil {
		return err
	}

	// route the case to any doctor assigned to the patient for this condition,
	// otherwise place in global unclaimed queue
	if careTeam != nil {
		for _, assignment := range careTeam.Assignments {
			if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.HealthConditionId == patientCase.HealthConditionId.Int64() {
				// we identified a doctor the case can be routed to
				if err := dataAPI.PermanentlyAssignDoctorToCaseAndRouteToQueue(assignment.ProviderId, patientCase, &api.DoctorQueueItem{
					DoctorId:  assignment.ProviderId,
					ItemId:    ev.VisitId,
					Status:    api.STATUS_PENDING,
					EventType: api.DQEventTypePatientVisit,
				}); err != nil {
					return err
				}
				return nil
			}
		}
	}

	// no doctor could be identified; place the case in the global queue
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

	return nil
}
