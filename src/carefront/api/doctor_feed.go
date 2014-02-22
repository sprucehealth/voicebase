package api

import (
	"carefront/settings"
	"fmt"
	"time"
)

const (
	EVENT_TYPE_PATIENT_VISIT            = "PATIENT_VISIT"
	EVENT_TYPE_TREATMENT_PLAN           = "TREATMENT_PLAN"
	EVENT_TYPE_REFILL_REQUEST           = "REFILL_REQUEST"
	patientVisitImageTag                = "patient_visit_queue_icon"
	beginPatientVisitReviewAction       = "begin_patient_visit"
	viewTreatedPatientVisitReviewAction = "view_treated_patient_visit"
	viewRefillRequestAction             = "view_refill_request"
)

type DoctorQueueItem struct {
	Id              int64
	DoctorId        int64
	EventType       string
	EnqueueDate     time.Time
	CompletedDate   time.Time
	ItemId          int64
	Status          string
	PositionInQueue int
}

func (d *DoctorQueueItem) GetTitleAndSubtitle(dataApi DataAPI) (string, string, error) {
	var title, subtitle string

	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		patientId, err := dataApi.GetPatientIdFromPatientVisitId(d.ItemId)
		if err != nil {
			return "", "", err
		}

		patient, err := dataApi.GetPatientFromId(patientId)
		if err != nil {
			return "", "", err
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
		case QUEUE_ITEM_STATUS_PHOTOS_REJECTED:
			title = fmt.Sprintf("Photos rejected for visit with %s %s", patient.FirstName, patient.LastName)
			formattedTime := d.EnqueueDate.Format("3:04pm")
			subtitle = fmt.Sprintf("%s %d at %s", d.EnqueueDate.Month().String(), d.EnqueueDate.Day(), formattedTime)
		case QUEUE_ITEM_STATUS_TRIAGED:
			title = fmt.Sprintf("Completed and triaged visit for %s %s", patient.FirstName, patient.LastName)
			formattedTime := d.EnqueueDate.Format("3:04pm")
			subtitle = fmt.Sprintf("%s %d at %s", d.EnqueueDate.Month().String(), d.EnqueueDate.Day(), formattedTime)
		}

	case EVENT_TYPE_REFILL_REQUEST:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err != nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("Refill request for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Continue refill request for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Refill request completed for %s %s", patient.FirstName, patient.LastName)
		}
	}
	return title, subtitle, nil
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
		return fmt.Sprintf("%s%s", SpruceImageBaseUrl, patientVisitImageTag)
	}
	return ""
}

func (d *DoctorQueueItem) GetDisplayTypes() []string {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT, EVENT_TYPE_TREATMENT_PLAN:
		switch d.Status {

		case QUEUE_ITEM_STATUS_PHOTOS_REJECTED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE}

		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}

		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue == 0 {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON}
			} else {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE}
			}
		}
	case EVENT_TYPE_REFILL_REQUEST:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue == 0 {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON}
			} else {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE}
			}

		}
	}
	return nil
}

func (d *DoctorQueueItem) GetActionUrl() string {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			return fmt.Sprintf("%s%s?patient_visit_id=%d", SpruceButtonBaseActionUrl, viewTreatedPatientVisitReviewAction, d.ItemId)
		}
	case EVENT_TYPE_TREATMENT_PLAN:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			return fmt.Sprintf("%s%s?treatment_plan_id=%d", SpruceButtonBaseActionUrl, viewTreatedPatientVisitReviewAction, d.ItemId)
		}
	case EVENT_TYPE_REFILL_REQUEST:
		switch d.Status {
		case QUEUE_ITEM_STATUS_ONGOING, QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_COMPLETED:
			return fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId)
		}
	}
	return ""
}

func (d *DoctorQueueItem) GetButton() *Button {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Begin"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", SpruceButtonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Continue"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", SpruceButtonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId)
			return button
		}
	case EVENT_TYPE_REFILL_REQUEST:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Begin"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Continue"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId)
			return button
		}
	}
	return nil
}
