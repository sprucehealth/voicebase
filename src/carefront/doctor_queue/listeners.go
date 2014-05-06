package doctor_queue

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/messages"
	"errors"
)

func InitListeners(dataAPI api.DataAPI) {
	dispatch.Default.Subscribe(func(ev *apiservice.VisitSubmittedEvent) error {
		// Insert into item appropriate doctor queue to make them aware of a new visit
		// for them to diagnose
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.VisitId,
			Status:    api.STATUS_PENDING,
			EventType: api.EVENT_TYPE_PATIENT_VISIT,
		}); err != nil {
			golog.Errorf("Unable to assign patient visit to doctor: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *apiservice.VisitReviewSubmittedEvent) error {
		// mark the status on the visit in the doctor's queue to move it to the completed tab
		// so that the visit is no longer in the hands of the doctor
		err := dataAPI.MarkGenerationOfTreatmentPlanInVisitQueue(ev.DoctorId,
			ev.VisitId, ev.TreatmentPlanId, api.QUEUE_ITEM_STATUS_ONGOING, ev.Status)
		if err != nil {
			golog.Errorf("Unable to update the status of the patient visit in the doctor queue: " + err.Error())
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
			eventTypeString = api.EVENT_TYPE_REFILL_TRANSMISSION_ERROR
		case common.UnlinkedDNTFTreatmentType:
			eventTypeString = api.EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR
		case common.ERxType:
			eventTypeString = api.EVENT_TYPE_TRANSMISSION_ERROR
		}
		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.ItemId,
			Status:    api.STATUS_PENDING,
			EventType: eventTypeString,
		}); err != nil {
			golog.Errorf("Unable to insert transmission error event into doctor queue: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *apiservice.RxTransmissionErrorResolvedEvent) error {
		// Insert item into appropriate doctor queue to indicate resolution of transmission error
		var eventType string
		switch ev.EventType {
		case common.ERxType:
			eventType = api.EVENT_TYPE_TRANSMISSION_ERROR
		case common.RefillRxType:
			eventType = api.EVENT_TYPE_REFILL_TRANSMISSION_ERROR
		case common.UnlinkedDNTFTreatmentType:
			eventType = api.EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR
		}
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.ItemId,
			EventType: eventType,
			Status:    api.QUEUE_ITEM_STATUS_COMPLETED,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
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
			EventType: api.EVENT_TYPE_REFILL_REQUEST,
			Status:    api.STATUS_PENDING,
		}); err != nil {
			golog.Errorf("Unable to insert refill request item into doctor queue: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *apiservice.RefillRequestResolvedEvent) error {
		// Move the queue item for the doctor from the ongoing to the completed state
		if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
			DoctorId:  ev.DoctorId,
			ItemId:    ev.RefillRequestId,
			EventType: api.EVENT_TYPE_REFILL_REQUEST,
			Status:    ev.Status,
		}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
			golog.Errorf("Unable to insert refill request resolved error into doctor queue: %s", err)
			return err
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationStartedEvent) error {
		people, err := dataAPI.GetPeople([]int64{ev.FromId, ev.ToId})
		if err != nil {
			return err
		}
		from := people[ev.FromId]
		if from == nil {
			return errors.New("failed to find person conversation is from")
		}
		to := people[ev.ToId]
		if to == nil {
			return errors.New("failed to find person conversation is addressed to")
		}

		// only act on event if the message goes from patient->doctor
		if to.RoleType != api.DOCTOR_ROLE || from.RoleType != api.PATIENT_ROLE {
			return nil
		}

		if err := dataAPI.InsertItemIntoDoctorQueue(api.DoctorQueueItem{
			DoctorId:  to.Doctor.DoctorId.Int64(),
			ItemId:    ev.ConversationId,
			EventType: api.EVENT_TYPE_CONVERSATION,
			Status:    api.STATUS_PENDING,
		}); err != nil {
			golog.Errorf("Unable to insert conversation item into doctor queue: %s", err)
			return err
		}
		return nil
	})
}
