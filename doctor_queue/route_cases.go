package doctor_queue

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/notify"
)

func routeIncomingPatientVisit(ev *cost.VisitChargedEvent, dataAPI api.DataAPI, notificationManager *notify.NotificationManager) error {

	// get the patient's care team
	careTeam, err := dataAPI.GetCareTeamForPatient(ev.PatientID)
	if err != nil {
		golog.Errorf("Unable to get care team for patient: %s", err)
		return err
	}

	// identify the MA and active doctor on the patient's care team
	var maID, activeDoctorID int64

	// get the patient case that the visit belongs to
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(ev.VisitID)
	if err != nil {
		golog.Errorf("Unable to get patient case from patient visit id: %s", err)
		return err
	}

	// route the case to any doctor assigned to the patient for this condition,
	// otherwise place in global unclaimed queue
	if careTeam != nil {
		for _, assignment := range careTeam.Assignments {
			if assignment.ProviderRole == api.DOCTOR_ROLE && assignment.HealthConditionId == patientCase.HealthConditionId.Int64() {
				activeDoctorID = assignment.ProviderID
			} else if assignment.ProviderRole == api.MA_ROLE && assignment.HealthConditionId == patientCase.HealthConditionId.Int64() {
				maID = assignment.ProviderID
			}
		}
	}

	// route the case to the active doctor already part of the patient's care team
	if activeDoctorID > 0 {
		if err := dataAPI.PermanentlyAssignDoctorToCaseAndRouteToQueue(activeDoctorID, patientCase, &api.DoctorQueueItem{
			DoctorId:  activeDoctorID,
			ItemId:    ev.VisitID,
			Status:    api.STATUS_PENDING,
			EventType: api.DQEventTypePatientVisit,
		}); err != nil {
			golog.Errorf("Unable to permanently assign doctor to case: %s", err)
			return err
		}

		if err := notifyMAOfCaseRoute(maID, ev, dataAPI, notificationManager); err != nil {
			golog.Errorf("Unable to notify MA of case route: %s", err)
		}

		return nil
	}

	// no doctor could be identified; place the case in the global queue
	// insert item into the unclaimed item queue given that it has not been claimed by a doctor yet
	patient, err := dataAPI.GetPatientFromId(ev.PatientID)
	if err != nil {
		golog.Errorf("Unable to get patient from id: %s", err)
		return err
	}

	careProvidingStateId, err := dataAPI.GetCareProvidingStateId(patient.StateFromZipCode, patientCase.HealthConditionId.Int64())
	if err != nil {
		golog.Errorf("Unable to get care providing state: %s", err)
		return err
	}

	if err := dataAPI.InsertUnclaimedItemIntoQueue(&api.DoctorQueueItem{
		CareProvidingStateId: careProvidingStateId,
		ItemId:               ev.VisitID,
		EventType:            api.DQEventTypePatientVisit,
		Status:               api.STATUS_PENDING,
		PatientCaseId:        patientCase.Id.Int64(),
	}); err != nil {
		golog.Errorf("Unable to insert case into unclaimed case queue: %s", err)
		return err
	}

	// also notify the MA of a visit submission so that the MA can be proactive in any communication with the patient
	if err := notifyMAOfCaseRoute(maID, ev, dataAPI, notificationManager); err != nil {
		golog.Errorf("unable to notify MA of case route: %s", err)
	}

	return nil
}

func notifyMAOfCaseRoute(maID int64, ev *cost.VisitChargedEvent, dataAPI api.DataAPI, notificationManager *notify.NotificationManager) error {
	// nothing to do as MA does not exist
	if maID == 0 {
		return nil
	}

	ma, err := dataAPI.GetDoctorFromId(maID)
	if err != nil {
		return err
	}

	return notificationManager.NotifyDoctor(api.MA_ROLE, ma.DoctorId.Int64(), ma.AccountId.Int64(), ev)
}
