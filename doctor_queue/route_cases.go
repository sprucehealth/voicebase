package doctor_queue

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/notify"
)

func routeIncomingPatientVisit(ev *cost.VisitChargedEvent, dataAPI api.DataAPI, notificationManager *notify.NotificationManager) error {

	// identify the MA and active doctor on the patient's care team
	var maID, activeDoctorID int64

	// get the patient case that the visit belongs to
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitID(ev.VisitID)
	if err != nil {
		golog.Errorf("Unable to get patient case from patient visit id: %s", err)
		return err
	}

	// get the members of the patient's care team
	members, err := dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
	if err != nil {
		golog.Errorf("Unable to get members of care team for case: %s", err.Error())
		return err
	}

	// route the case to any doctor assigned to the patient case,
	// otherwise place in global unclaimed queue
	for _, assignment := range members {
		switch assignment.ProviderRole {
		case api.DOCTOR_ROLE:
			activeDoctorID = assignment.ProviderID
		case api.MA_ROLE:
			maID = assignment.ProviderID
		}
	}

	// no doctor could be identified; place the case in the global queue
	// insert item into the unclaimed item queue given that it has not been claimed by a doctor yet
	patient, err := dataAPI.GetPatientFromID(ev.PatientID)
	if err != nil {
		golog.Errorf("Unable to get patient from id: %s", err)
		return err
	}

	// route the case to the active doctor already part of the patient's care team
	if activeDoctorID > 0 {

		var description, shortDescription, notifyMessage string
		if ev.IsFollowup {
			description = fmt.Sprintf("Follow-up visit for %s %s", patient.FirstName, patient.LastName)
			shortDescription = "Follow-up visit"
			notifyMessage = "One of your Spruce patients just completed a follow-up visit."
		} else {
			description = fmt.Sprintf("New visit for %s %s", patient.FirstName, patient.LastName)
			shortDescription = "New visit"
			notifyMessage = "A new Spruce patient case has been assigned to you."
		}

		if err := dataAPI.PermanentlyAssignDoctorToCaseAndRouteToQueue(activeDoctorID, patientCase, &api.DoctorQueueItem{
			DoctorID:         activeDoctorID,
			PatientID:        patient.PatientID.Int64(),
			ItemID:           ev.VisitID,
			Status:           api.STATUS_PENDING,
			EventType:        api.DQEventTypePatientVisit,
			Description:      description,
			ShortDescription: shortDescription,
			ActionURL:        app_url.ViewPatientVisitInfoAction(patient.PatientID.Int64(), ev.VisitID, patientCase.ID.Int64()),
			Tags:             []string{patientCase.Name},
		}); err != nil {
			golog.Errorf("Unable to permanently assign doctor to case: %s", err)
			return err
		}

		if err := notifyMAOfCaseRoute(maID, ev, dataAPI, notificationManager); err != nil {
			golog.Errorf("Unable to notify MA of case route: %s", err)
		}

		// notify the doctor of the case route
		accountID, err := dataAPI.GetAccountIDFromDoctorID(activeDoctorID)
		if err != nil {
			golog.Errorf(err.Error())
			return err
		}

		if err := notificationManager.NotifyDoctor(
			api.DOCTOR_ROLE,
			activeDoctorID,
			accountID,
			&notify.Message{
				ShortMessage: notifyMessage,
			}); err != nil {
			golog.Errorf(err.Error())
			return err
		}

		return nil
	}

	careProvidingStateID, err := dataAPI.GetCareProvidingStateID(patient.StateFromZipCode, patientCase.PathwayTag)
	if err != nil {
		golog.Errorf("Unable to get care providing state: %s", err)
		return err
	}

	if err := dataAPI.InsertUnclaimedItemIntoQueue(&api.DoctorQueueItem{
		CareProvidingStateID: careProvidingStateID,
		PatientID:            patient.PatientID.Int64(),
		ItemID:               ev.VisitID,
		EventType:            api.DQEventTypePatientVisit,
		Status:               api.STATUS_PENDING,
		PatientCaseID:        patientCase.ID.Int64(),
		Description:          fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName),
		ShortDescription:     "New visit",
		ActionURL:            app_url.ViewPatientVisitInfoAction(patient.PatientID.Int64(), ev.VisitID, patientCase.ID.Int64()),
		Tags:                 []string{patientCase.Name},
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

	ma, err := dataAPI.GetDoctorFromID(maID)
	if err != nil {
		return err
	}

	return notificationManager.NotifyDoctor(
		api.MA_ROLE,
		ma.DoctorID.Int64(),
		ma.AccountID.Int64(),
		&notify.Message{
			ShortMessage: "A patient has submitted a Spruce visit.",
		})
}
