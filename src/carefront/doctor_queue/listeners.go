package doctor_queue

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/app_worker"
	"carefront/common"
	"carefront/doctor_treatment_plan"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/messages"
	"carefront/notify"
	"carefront/patient_visit"
	"errors"
)

func InitListeners(dataAPI api.DataAPI, notificationManager *notify.NotificationManager) {
	dispatch.Default.Subscribe(func(ev *patient_visit.VisitSubmittedEvent) error {
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

	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanCreatedEvent) error {
		// mark the status on the visit in the doctor's queue to move it to the completed tab
		// so that the visit is no longer in the hands of the doctor
		err := dataAPI.MarkGenerationOfTreatmentPlanInVisitQueue(ev.DoctorId,
			ev.VisitId, ev.TreatmentPlanId, api.QUEUE_ITEM_STATUS_ONGOING, api.CASE_STATUS_TREATED)
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
			EventType: api.EVENT_TYPE_PATIENT_VISIT,
			Status:    api.QUEUE_ITEM_STATUS_TRIAGED,
		}, api.QUEUE_ITEM_STATUS_ONGOING); err != nil {
			golog.Errorf("Unable to insert transmission error resolved into doctor queue: %s", err)
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

		if err := notificationManager.NotifyDoctor(to.Doctor, ev); err != nil {
			golog.Errorf("Unable to notify doctor: %s", err)
			return err
		}

		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationReplyEvent) error {
		conversation, err := dataAPI.GetConversation(ev.ConversationId)
		if err != nil {
			return err
		}

		// clear the item from the doctor's queue once they respond to a message
		person := conversation.Participants[ev.FromId]
		if person.RoleType == api.DOCTOR_ROLE {
			if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
				DoctorId:  person.Doctor.DoctorId.Int64(),
				ItemId:    ev.ConversationId,
				EventType: api.EVENT_TYPE_CONVERSATION,
				Status:    api.QUEUE_ITEM_STATUS_REPLIED,
			}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
				golog.Errorf("Unable to replace item in doctor queue with a replied item: %s", err)
				return err
			}
		}

		// if in the event the patient initiates the reply, refresh an existing item in the doctor's queue
		if person.RoleType == api.PATIENT_ROLE {
			// find the doctor to insert item into their queue
			for _, p := range conversation.Participants {
				if p.RoleType == api.DOCTOR_ROLE {
					if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
						DoctorId:  p.Doctor.DoctorId.Int64(),
						ItemId:    ev.ConversationId,
						EventType: api.EVENT_TYPE_CONVERSATION,
						Status:    api.QUEUE_ITEM_STATUS_PENDING,
					}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
						golog.Errorf("Unable to replace item in doctor queue with a replied item: %s", err)
						return err
					}

					if err := notificationManager.NotifyDoctor(p.Doctor, ev); err != nil {
						golog.Errorf("Unable to notify doctor: %s", err)
						return err
					}
				}
			}
		}
		return nil
	})

	dispatch.Default.Subscribe(func(ev *messages.ConversationReadEvent) error {
		// delete the item from the queue when the doctor marks the conversation
		// as being read
		people, err := dataAPI.GetPeople([]int64{ev.FromId})
		if err != nil {
			return err
		}

		person := people[ev.FromId]
		if person.RoleType == api.DOCTOR_ROLE {
			if err := dataAPI.ReplaceItemInDoctorQueue(api.DoctorQueueItem{
				DoctorId:  person.Doctor.DoctorId.Int64(),
				ItemId:    ev.ConversationId,
				EventType: api.EVENT_TYPE_CONVERSATION,
				Status:    api.QUEUE_ITEM_STATUS_READ,
			}, api.QUEUE_ITEM_STATUS_PENDING); err != nil {
				golog.Errorf("Unable to replace item in doctor queue with a replied item: %s", err)
				return err
			}
		}
		return nil
	})

}
