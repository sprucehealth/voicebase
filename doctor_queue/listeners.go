package doctor_queue

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_event"
	"github.com/sprucehealth/backend/app_worker"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/notify"
	"github.com/sprucehealth/backend/patient_visit"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func InitListeners(dataAPI api.DataAPI, notificationManager *notify.NotificationManager, statsRegistry metrics.Registry, jbcqMinutesThreshold int) {
	initJumpBallCaseQueueListeners(dataAPI, statsRegistry, jbcqMinutesThreshold)

	routeSuccess := metrics.NewCounter()
	routeFailure := metrics.NewCounter()
	statsRegistry.Add("route/success", routeSuccess)
	statsRegistry.Add("route/failure", routeFailure)

	dispatch.Default.Subscribe(func(ev *patient_visit.VisitSubmittedEvent) error {
		// route the incoming visit to a doctor queue
		if err := routeIncomingPatientVisit(ev, dataAPI); err != nil {
			routeFailure.Inc(1)
			golog.Errorf("Unable to route incoming patient visit: %s", err)
			return err
		}
		routeSuccess.Inc(1)
		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		// mark the status on the visit in the doctor's queue to move it to the completed tab
		// so that the visit is no longer in the hands of the doctor
		err := dataAPI.MarkGenerationOfTreatmentPlanInVisitQueue(ev.DoctorId,
			ev.VisitId, ev.TreatmentPlan.Id.Int64(), api.DQItemStatusOngoing, api.DQItemStatusTreated)
		if err != nil {
			golog.Errorf("Unable to update the status of the patient visit in the doctor queue: " + err.Error())
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		// mark the visit as complete once the doctor submits a diagnosis to indicate that the
		// patient was unsuitable for spruce
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.PatientVisitId,
			EventType: api.DQEventTypePatientVisit,
			Status:    api.DQItemStatusTriaged,
		}, api.DQItemStatusOngoing); err != nil {
			golog.Errorf("Unable to insert transmission error resolved into doctor queue: %s", err)
			return err
		}

		// assign the case to the MA
		assignments, err := dataAPI.GetActiveMembersOfCareTeamForCase(ev.CaseID, false)
		if err != nil {
			golog.Errorf("Unable to get active members of care team for case: %s", err)
			return err
		}

		var maID int64
		for _, assignment := range assignments {
			if assignment.ProviderRole == api.MA_ROLE {
				maID = assignment.ProviderID
				break
			}
		}
		ma, err := dataAPI.GetDoctorFromId(maID)
		if err != nil {
			golog.Errorf("Unable to get MA from id: %s", err)
			return err
		}
		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			golog.Errorf("Unable to get doctor from id: %s", err)
			return err
		}

		message := &common.CaseMessage{
			CaseID:    ev.CaseID,
			PersonID:  doctor.PersonId,
			Body:      fmt.Sprintf(`Case was marked as unsuitable for spruce with following explanation from the doctor:"%s"`, ev.InternalReason),
			IsPrivate: true,
			EventText: fmt.Sprintf("assigned to %s", ma.LongDisplayName),
		}

		if _, err := dataAPI.CreateCaseMessage(message); err != nil {
			golog.Errorf("Unable to create private message to assign case to MA : %s", err)
			return err
		}

		// insert a pending item into the MA's queue
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ma.DoctorId.Int64(),
			ItemId:    ev.CaseID,
			EventType: api.DQEventTypeCaseAssignment,
			Status:    api.DQItemStatusPending,
		}); err != nil {
			golog.Errorf("Unable to insert case assignment item into doctor queue: %s", err)
			routeFailure.Inc(1)
			return err
		}

		// notify the ma of the case assignment
		if err := notificationManager.NotifyDoctor(ma, ev); err != nil {
			golog.Errorf("Unable to notify assigned provider of event %T: %s", ev, err)
			routeFailure.Inc(1)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *app_worker.RxTransmissionErrorEvent) error {
		// Insert item into appropriate doctor queue to make them ever of an erx
		// that had issues being routed to pharmacy
		var eventTypeString string
		switch ev.EventType {
		case common.RefillRxType:
			eventTypeString = api.DQEventTypeRefillTransmissionError
		case common.UnlinkedDNTFTreatmentType:
			eventTypeString = api.DQEventTypeUnlinkedDNTFTransmissionError
		case common.ERxType:
			eventTypeString = api.DQEventTypeTransmissionError
		}
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.ItemId,
			Status:    api.STATUS_PENDING,
			EventType: eventTypeString,
		}); err != nil {
			routeFailure.Inc(1)
			golog.Errorf("Unable to insert transmission error event into doctor queue: %s", err)
			return err
		}
		routeSuccess.Inc(1)

		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			golog.Errorf("Unable to get doctor from id: %s", err)
			return err
		}

		if err := notificationManager.NotifyDoctor(doctor, ev); err != nil {
			golog.Errorf("Unable to notify doctor: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor.RxTransmissionErrorResolvedEvent) error {
		// Insert item into appropriate doctor queue to indicate resolution of transmission error
		var eventType string
		switch ev.EventType {
		case common.ERxType:
			eventType = api.DQEventTypeTransmissionError
		case common.RefillRxType:
			eventType = api.DQEventTypeRefillTransmissionError
		case common.UnlinkedDNTFTreatmentType:
			eventType = api.DQEventTypeUnlinkedDNTFTransmissionError
		}
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.ItemId,
			EventType: eventType,
			Status:    api.DQItemStatusTreated,
		}, api.DQItemStatusPending); err != nil {
			golog.Errorf("Unable to insert transmission error resolved into doctor queue: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *app_worker.RefillRequestCreatedEvent) error {
		// insert refill item into doctor queue as a refill request
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.RefillRequestId,
			EventType: api.DQEventTypeRefillRequest,
			Status:    api.STATUS_PENDING,
		}); err != nil {
			routeFailure.Inc(1)
			golog.Errorf("Unable to insert refill request item into doctor queue: %s", err)
			return err
		}
		routeSuccess.Inc(1)

		doctor, err := dataAPI.GetDoctorFromId(ev.DoctorId)
		if err != nil {
			golog.Errorf("Unable to get doctor from id: %s", err)
			return err
		}

		if err := notificationManager.NotifyDoctor(doctor, ev); err != nil {
			golog.Errorf("Unable to notify doctor: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *doctor.RefillRequestResolvedEvent) error {
		// Move the queue item for the doctor from the ongoing to the completed state
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.RefillRequestId,
			EventType: api.DQEventTypeRefillRequest,
			Status:    ev.Status,
		}, api.DQItemStatusPending); err != nil {
			golog.Errorf("Unable to insert refill request resolved error into doctor queue: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.PostEvent) error {
		// clear the item from the doctor's queue once they respond to a message
		if ev.Person.RoleType == api.DOCTOR_ROLE {
			if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
				DoctorId:  ev.Person.RoleId,
				ItemId:    ev.Case.Id.Int64(),
				EventType: api.DQEventTypeCaseMessage,
				Status:    api.DQItemStatusReplied,
			}, api.DQItemStatusPending); err != nil {
				golog.Errorf("Unable to replace item in doctor queue with a replied item: %s", err)
				return err
			}

			return nil
		}

		// only act on event if the message goes from patient->doctor
		if ev.Person.RoleType != api.PATIENT_ROLE {
			return nil
		}

		// send the patient message to the MA, or to the doctor if the MA doesn't exist
		assignments, err := dataAPI.GetActiveMembersOfCareTeamForCase(ev.Case.Id.Int64(), false)
		if err != nil {
			golog.Errorf("Unable to get doctors assignend to case: %s", err)
			return err
		}

		var doctorID, maID, providerToAssignToID int64
		for _, assignment := range assignments {
			switch assignment.Status {
			case api.STATUS_ACTIVE:
				if assignment.ProviderRole == api.DOCTOR_ROLE {
					doctorID = assignment.ProviderID
				} else if assignment.ProviderRole == api.MA_ROLE {
					maID = assignment.ProviderID
				}
			}
		}

		if doctorID == 0 && maID == 0 {
			// No doctor or ma assigned to patient
			return errors.New("No doctor or ma assigned to patient case")
		} else if maID > 0 {
			providerToAssignToID = maID
		} else {
			providerToAssignToID = doctorID
		}

		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  providerToAssignToID,
			ItemId:    ev.Message.CaseID,
			EventType: api.DQEventTypeCaseMessage,
			Status:    api.DQItemStatusPending,
		}); err != nil {
			routeFailure.Inc(1)
			golog.Errorf("Unable to insert conversation item into doctor queue: %s", err)
			return err
		}
		routeSuccess.Inc(1)

		doctor, err := dataAPI.GetDoctorFromId(providerToAssignToID)
		if err != nil {
			return err
		}

		if err := notificationManager.NotifyDoctor(doctor, ev); err != nil {
			golog.Errorf("Unable to notify doctor: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.CaseAssignEvent) error {

		// create an item in the history tab for the provider assigning the case
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.Person.RoleId,
			ItemId:    ev.Case.Id.Int64(),
			EventType: api.DQEventTypeCaseAssignment,
			Status:    api.DQItemStatusReplied,
		}, api.DQItemStatusPending); err != nil {
			golog.Errorf("Unable to insert case assignment item into doctor queue: %s", err)
			routeFailure.Inc(1)
			return err
		}

		// identify the provider the case is being assigned to
		var assignedProvider *common.Doctor
		if ev.Person.RoleType == api.DOCTOR_ROLE {
			assignedProvider = ev.MA
		} else {
			assignedProvider = ev.Doctor
		}

		// insert a pending item into the queue of the assigned provider
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  assignedProvider.DoctorId.Int64(),
			ItemId:    ev.Case.Id.Int64(),
			EventType: api.DQEventTypeCaseAssignment,
			Status:    api.DQItemStatusPending,
		}); err != nil {
			golog.Errorf("Unable to insert case assignment item into doctor queue: %s", err)
			routeFailure.Inc(1)
			return err
		}

		// notify the assigned provider
		if err := notificationManager.NotifyDoctor(assignedProvider, ev); err != nil {
			golog.Errorf("Unable to notify assigned provider of event %T: %s", ev, err)
			routeFailure.Inc(1)
			return err
		}

		routeSuccess.Inc(1)
		return nil
	})

	dispatch.Default.Subscribe(func(ev *app_event.AppEvent) error {

		// delete the item from the queue when the doctor marks the conversation
		// as being read
		if ev.Resource == "all_case_messages" && ev.Action == app_event.ViewedAction {
			switch ev.Role {
			case api.DOCTOR_ROLE, api.MA_ROLE:
			default:
				return nil
			}

			doctorId, err := dataAPI.GetDoctorIdFromAccountId(ev.AccountId)
			if err != nil {
				golog.Errorf("Unable to get doctor id from account id: %s", err)
				return err
			}

			if err := dataAPI.DeleteItemFromDoctorQueue(api.DoctorQueueItem{
				DoctorId:  doctorId,
				ItemId:    ev.ResourceId,
				EventType: api.DQEventTypeCaseMessage,
				Status:    api.DQItemStatusPending,
			}); err != nil {
				golog.Errorf("Unable to delete pending message item in doctor queue :%s", err)
			}

			if err := dataAPI.DeleteItemFromDoctorQueue(api.DoctorQueueItem{
				DoctorId:  doctorId,
				ItemId:    ev.ResourceId,
				EventType: api.DQEventTypeCaseAssignment,
				Status:    api.DQItemStatusPending,
			}); err != nil {
				golog.Errorf("Unable to delete pending assignment item in doctor queue :%s", err)
			}
		}

		return nil
	})
}
