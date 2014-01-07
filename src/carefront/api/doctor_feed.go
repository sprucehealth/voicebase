package api

import (
	"fmt"
	"time"
)

const (
	EVENT_TYPE_PATIENT_VISIT = "PATIENT_VISIT"
	patientVisitImageTag     = "patient_visit_queue_icon"
	buttonBaseActionUrl      = "spruce:///action/"
)

type DoctorQueueItem struct {
	Id            int64
	DoctorId      int64
	EventType     string
	EnqueueDate   time.Time
	CompletedDate time.Time
	ItemId        int64
	Status        string
}

func (d *DoctorQueueItem) GetTitleAndSubtitle(dataApi DataAPI) (title, subtitle string, err error) {
	switch d.EventType {

	case EVENT_TYPE_PATIENT_VISIT:
		patientId, shadowedErr := dataApi.GetPatientIdFromPatientVisitId(d.ItemId)
		if shadowedErr != nil {
			err = shadowedErr
			return
		}

		patient, shadowedErr := dataApi.GetPatientFromId(patientId)
		if shadowedErr != nil {
			err = shadowedErr
			return
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Treatment Plan completed for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName)
		}
	}
	return
}

func (d *DoctorQueueItem) GetImageTag() string {
	return patientVisitImageTag
}

func (d *DoctorQueueItem) GetDisplayTypes() []string {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE}
		case QUEUE_ITEM_STATUS_PENDING:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON}
		}
	}
	return nil
}

func (d *DoctorQueueItem) GetButton() *Button {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED:
			return nil
		case QUEUE_ITEM_STATUS_PENDING:
			button := &Button{}
			button.ButtonText = "Begin"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", buttonBaseActionUrl, "begin_patient_visit", d.ItemId)
			return button
		}
	}
	return nil
}
