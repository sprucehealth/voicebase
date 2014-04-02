package api

import (
	"carefront/settings"
	"fmt"
	"time"
)

const (
	EVENT_TYPE_PATIENT_VISIT                    = "PATIENT_VISIT"
	EVENT_TYPE_TREATMENT_PLAN                   = "TREATMENT_PLAN"
	EVENT_TYPE_REFILL_REQUEST                   = "REFILL_REQUEST"
	EVENT_TYPE_TRANSMISSION_ERROR               = "TRANSMISSION_ERROR"
	EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR = "UNLINKED_DNTF_TRANSMISSION_ERROR"
	EVENT_TYPE_REFILL_TRANSMISSION_ERROR        = "REFILL_TRANSMISSION_ERROR"
	patientVisitImageTag                        = "patient_visit_queue_icon"
	beginPatientVisitReviewAction               = "begin_patient_visit"
	viewTreatedPatientVisitReviewAction         = "view_treated_patient_visit"
	viewRefillRequestAction                     = "view_refill_request"
	viewTransmissionErrorAction                 = "view_transmission_error"
	viewPatientTreatmentsAction                 = "view_patient_treatments"
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
	case EVENT_TYPE_PATIENT_VISIT, EVENT_TYPE_TREATMENT_PLAN:
		var patientVisitId int64
		var err error

		if d.EventType == EVENT_TYPE_TREATMENT_PLAN {
			patientVisitId, err = dataApi.GetPatientVisitIdFromTreatmentPlanId(d.ItemId)
			if err != nil {
				return "", "", err
			}
		} else {
			patientVisitId = d.ItemId
		}

		patientId, err := dataApi.GetPatientIdFromPatientVisitId(patientVisitId)
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
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName)
			subtitle = getRemainingTimeSubtitleForCaseToBeReviewed(d.EnqueueDate)
		case QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName)
			subtitle = getRemainingTimeSubtitleForCaseToBeReviewed(d.EnqueueDate)
		case QUEUE_ITEM_STATUS_PHOTOS_REJECTED:
			title = fmt.Sprintf("Photos rejected for visit with %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_TRIAGED:
			title = fmt.Sprintf("Completed and triaged visit for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_REFILL_REQUEST:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err != nil || patient == nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("Refill request for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_REFILL_APPROVED:
			title = fmt.Sprintf("Refill request approved for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_REFILL_DENIED:
			title = fmt.Sprintf("Refill request denied for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_REFILL_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err != nil {
			return "", "", err
		}

		if patient == nil {
			return "", "", nil
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("Error completing refill request for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Refill request error resolved for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_TRANSMISSION_ERROR, EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromTreatmentId(d.ItemId)
		if err != nil || patient == nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Error sending prescription for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Error resolved for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		unlinkedTreatment, err := dataApi.GetUnlinkedDNTFTreatment(d.ItemId)
		if err != nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Error sending prescription for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Error resolved for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
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

func (d *DoctorQueueItem) GetTimestamp() int64 {
	return d.EnqueueDate.Unix()
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
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
			}
		}
	case EVENT_TYPE_REFILL_REQUEST, EVENT_TYPE_REFILL_TRANSMISSION_ERROR:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue == 0 {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON}
			} else {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
			}
		case QUEUE_ITEM_STATUS_REFILL_APPROVED, QUEUE_ITEM_STATUS_REFILL_DENIED, QUEUE_ITEM_STATUS_COMPLETED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
		}
	case EVENT_TYPE_TRANSMISSION_ERROR, EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue == 0 {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_BUTTON}
			} else {
				return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
			}
		case QUEUE_ITEM_STATUS_COMPLETED:
			return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
		}
	}
	return nil
}

func (d *DoctorQueueItem) GetActionUrl(dataApi DataAPI) (string, error) {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			patientId, err := dataApi.GetPatientIdFromPatientVisitId(d.ItemId)
			if err != nil {
				return "", err
			}

			patient, err := dataApi.GetPatientFromId(patientId)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s%s?patient_id=%d", SpruceButtonBaseActionUrl, viewPatientTreatmentsAction, patient.PatientId.Int64()), nil
		case QUEUE_ITEM_STATUS_ONGOING, QUEUE_ITEM_STATUS_PENDING:
			return fmt.Sprintf("%s%s?patient_visit_id=%d", SpruceButtonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId), nil
		}
	case EVENT_TYPE_TREATMENT_PLAN:

	case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
		patientVisitId, err := dataApi.GetPatientVisitIdFromTreatmentPlanId(d.ItemId)
		if err != nil {
			return "", err
		}

		patientId, err := dataApi.GetPatientIdFromPatientVisitId(patientVisitId)
		if err != nil {
			return "", err
		}

		patient, err := dataApi.GetPatientFromId(patientId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s?patient_id=%d", SpruceButtonBaseActionUrl, viewPatientTreatmentsAction, patient.PatientId.Int64()), nil
	case EVENT_TYPE_REFILL_REQUEST, EVENT_TYPE_REFILL_TRANSMISSION_ERROR:
		return fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId), nil
	case EVENT_TYPE_TRANSMISSION_ERROR:
		return fmt.Sprintf("%s%s?treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId), nil
	case EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		return fmt.Sprintf("%s%s?unlinked_dntf_treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId), nil
	}
	return "", nil
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
			button := &Button{}
			button.ButtonText = "Continue"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?patient_visit_id=%d", SpruceButtonBaseActionUrl, beginPatientVisitReviewAction, d.ItemId)
			return button
		}
	case EVENT_TYPE_REFILL_REQUEST, EVENT_TYPE_REFILL_TRANSMISSION_ERROR:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			if d.EventType == EVENT_TYPE_REFILL_TRANSMISSION_ERROR {
				button.ButtonText = "Resolve Error"
			} else {
				button.ButtonText = "Begin"
			}
			button.ButtonActionUrl = fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			if d.EventType == EVENT_TYPE_REFILL_TRANSMISSION_ERROR {
				button.ButtonText = "Resolve Error"
			} else {
				button.ButtonText = "Continue"
			}
			button.ButtonActionUrl = fmt.Sprintf("%s%s?refill_request_id=%d", SpruceButtonBaseActionUrl, viewRefillRequestAction, d.ItemId)
			return button
		}
	case EVENT_TYPE_TRANSMISSION_ERROR:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Resolve Error"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Resolve Error"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId)
			return button
		}
	case EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Resolve Error"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?unlinked_dntf_treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId)
			return button
		case QUEUE_ITEM_STATUS_ONGOING:
			if d.PositionInQueue != 0 {
				return nil
			}
			button := &Button{}
			button.ButtonText = "Resolve Error"
			button.ButtonActionUrl = fmt.Sprintf("%s%s?unlinked_dntf_treatment_id=%d", SpruceButtonBaseActionUrl, viewTransmissionErrorAction, d.ItemId)
			return button
		}
	}
	return nil
}
