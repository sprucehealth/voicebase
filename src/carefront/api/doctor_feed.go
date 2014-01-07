package api

import (
	"carefront/settings"
	"fmt"
	"time"
)

const (
	EVENT_TYPE_PATIENT_VISIT      = "PATIENT_VISIT"
	patientVisitImageTag          = "patient_visit_queue_icon"
	buttonBaseActionUrl           = "spruce:///action/"
	imageBaseUrl                  = "spruce:///image/"
	beginPatientVisitReviewAction = "begin_patient_visit"
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
			formattedTime := d.EnqueueDate.Format("3:04pm")
			subtitle = fmt.Sprintf("%s %d at %s", d.EnqueueDate.Month().String(), d.EnqueueDate.Day(), formattedTime)
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName)
			subtitle = getRemainingTimeSubtitleForCaseToBeReviewed(d.EnqueueDate)
		case QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName)
			subtitle = getRemainingTimeSubtitleForCaseToBeReviewed(d.EnqueueDate)
		}
	}
	return
}

func getRemainingTimeSubtitleForCaseToBeReviewed(enqueueDate time.Time) string {
	timeLeft := enqueueDate.Add(settings.SLA_TO_SERVICE_CUSTOMER).Sub(time.Now())
	minutesLeft := int64(timeLeft.Minutes()) - (60 * int64(timeLeft.Hours()))
	subtitle := fmt.Sprintf("%dh %dm left", int64(timeLeft.Hours()), int64(minutesLeft))
	return subtitle
}

func (d *DoctorQueueItem) GetImageUrl() string {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		return fmt.Sprintf("%s%s", imageBaseUrl, patientVisitImageTag)
	}
	return ""
}

func (d *DoctorQueueItem) GetDisplayTypes() []string {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE}
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
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
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", buttonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			button := &Button{}
			button.ButtonText = "Continue"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", buttonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId)
			return button
		}
	}
	return nil
}
