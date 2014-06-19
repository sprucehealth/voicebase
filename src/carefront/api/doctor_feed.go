package api

import (
	"carefront/app_url"
	"carefront/common"
	"carefront/libs/golog"
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
	EVENT_TYPE_CASE_MESSAGE                     = "CASE_MESSAGE"
)

type DoctorQueueItem struct {
	Id                   int64
	DoctorId             int64
	EventType            string
	EnqueueDate          time.Time
	CompletedDate        time.Time
	ItemId               int64
	Status               string
	PositionInQueue      int
	CareProvidingStateId int64
}

func (d *DoctorQueueItem) GetTitleAndSubtitle(dataApi DataAPI) (string, string, error) {
	var title, subtitle string

	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT, EVENT_TYPE_TREATMENT_PLAN:
		var patient *common.Patient
		var err error

		if d.EventType == EVENT_TYPE_TREATMENT_PLAN {
			patient, err = dataApi.GetPatientFromTreatmentPlanId(d.ItemId)
			if err == NoRowsError {
				golog.Errorf("Did not get patient from treatment plan id (%d)", d.ItemId)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
		} else {
			patient, err = dataApi.GetPatientFromPatientVisitId(d.ItemId)
			if err == NoRowsError {
				golog.Errorf("Did not get patient from patient visit id (%d)", d.ItemId)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
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
		case QUEUE_ITEM_STATUS_TRIAGED:
			title = fmt.Sprintf("Completed and triaged visit for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_REFILL_REQUEST:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemId)
			return "", "", nil
		} else if err != nil {
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
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemId)
			return "", "", nil
		} else if err != nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING:
			title = fmt.Sprintf("Error completing refill request for %s %s", patient.FirstName, patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Refill request error resolved for %s %s", patient.FirstName, patient.LastName)
		}

	case EVENT_TYPE_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromTreatmentId(d.ItemId)
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from treatment id %d", d.ItemId)
			return "", "", nil
		} else if err != nil {
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
		if err == NoRowsError {
			golog.Errorf("Unable to get unlinked dntf treatment from id %d", d.ItemId)
			return "", "", nil
		} else if err != nil {
			return "", "", err
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_PENDING, QUEUE_ITEM_STATUS_ONGOING:
			title = fmt.Sprintf("Error sending prescription for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
		case QUEUE_ITEM_STATUS_COMPLETED:
			title = fmt.Sprintf("Error resolved for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
		}
	case EVENT_TYPE_CASE_MESSAGE:
		participants, err := dataApi.CaseMessageParticipants(d.ItemId, true)
		if err != nil {
			return "", "", err
		}
		for _, par := range participants {
			person := par.Person
			if person.RoleType == PATIENT_ROLE {
				patient := person.Patient
				switch d.Status {
				case QUEUE_ITEM_STATUS_PENDING:
					title = fmt.Sprintf("Message from %s %s", patient.FirstName, patient.LastName)
				case QUEUE_ITEM_STATUS_READ:
					title = fmt.Sprintf("Conversation with %s %s", patient.FirstName, patient.LastName)
				case QUEUE_ITEM_STATUS_REPLIED:
					title = fmt.Sprintf("Replied to %s %s", patient.FirstName, patient.LastName)
				}
				break
			}
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

func (d *DoctorQueueItem) GetImageUrl() *app_url.SpruceAsset {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		return app_url.PatientVisitQueueIcon
	}
	return nil
}

func (d *DoctorQueueItem) GetTimestamp() *time.Time {
	if d.EnqueueDate.IsZero() {
		return nil
	}

	return &d.EnqueueDate
}

func (d *DoctorQueueItem) GetDisplayTypes() []string {
	return []string{DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE}
}

func (d *DoctorQueueItem) ActionUrl(dataApi DataAPI) (*app_url.SpruceAction, error) {
	switch d.EventType {
	case EVENT_TYPE_PATIENT_VISIT:
		patientId, err := dataApi.GetPatientIdFromPatientVisitId(d.ItemId)
		if err != nil {
			golog.Errorf("Unable to get patient id from patient visit id: %s", err)
			return nil, nil
		}
		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			return app_url.ViewCompletedPatientVisitAction(patientId, d.ItemId), nil
		case QUEUE_ITEM_STATUS_ONGOING, QUEUE_ITEM_STATUS_PENDING:
			return app_url.BeginPatientVisitReviewAction(patientId, d.ItemId), nil
		}
	case EVENT_TYPE_TREATMENT_PLAN:

		switch d.Status {
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_TRIAGED:
			patientVisitId, err := dataApi.GetPatientVisitIdFromTreatmentPlanId(d.ItemId)

			if err == NoRowsError {
				golog.Errorf("Unable to get patient visit id from treatment plan id %d", d.ItemId)
				return nil, nil
			} else if err != nil {
				return nil, err
			}

			patientId, err := dataApi.GetPatientIdFromPatientVisitId(patientVisitId)
			if err != nil {
				golog.Errorf("Unable to get patient id from patient visit id: %s", err)
				return nil, nil
			}

			return app_url.ViewCompletedPatientVisitAction(patientId, patientVisitId), nil
		}
	case EVENT_TYPE_REFILL_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id: %s", err)
			return nil, nil
		}

		return app_url.ViewRefillRequestAction(patient.PatientId.Int64(), d.ItemId), nil
	case EVENT_TYPE_REFILL_REQUEST:
		patient, err := dataApi.GetPatientFromRefillRequestId(d.ItemId)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemId)
			return nil, nil
		}

		switch d.Status {
		case QUEUE_ITEM_STATUS_ONGOING, QUEUE_ITEM_STATUS_PENDING:
			return app_url.ViewRefillRequestAction(patient.PatientId.Int64(), d.ItemId), nil
		case QUEUE_ITEM_STATUS_COMPLETED, QUEUE_ITEM_STATUS_REFILL_APPROVED, QUEUE_ITEM_STATUS_REFILL_DENIED:
			return app_url.ViewPatientTreatmentsAction(patient.PatientId.Int64()), nil
		}
	case EVENT_TYPE_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromTreatmentId(d.ItemId)
		if err != nil {
			golog.Errorf("Unable to get patient from treatment id : %s", err)
			return nil, nil
		}
		return app_url.ViewTransmissionErrorAction(patient.PatientId.Int64(), d.ItemId), nil
	case EVENT_TYPE_UNLINKED_DNTF_TRANSMISSION_ERROR:
		patient, err := dataApi.GetPatientFromUnlinkedDNTFTreatment(d.ItemId)
		if err != nil {
			golog.Errorf("Unable to get patient from unlinked dntf treatment: %s", err)
			return nil, nil
		}
		return app_url.ViewTransmissionErrorAction(patient.PatientId.Int64(), d.ItemId), nil
	case EVENT_TYPE_CASE_MESSAGE:
		participants, err := dataApi.CaseMessageParticipants(d.ItemId, false)
		if err != nil {
			return nil, err
		}
		for _, p := range participants {
			if p.Person.RoleType == PATIENT_ROLE {
				return app_url.ViewPatientConversationsAction(p.Person.RoleId, d.ItemId), nil
			}
		}
	}

	return nil, nil
}
