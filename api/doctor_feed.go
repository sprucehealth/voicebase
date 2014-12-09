package api

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	DQEventTypePatientVisit                  = "PATIENT_VISIT"
	DQEventTypeTreatmentPlan                 = "TREATMENT_PLAN"
	DQEventTypeRefillRequest                 = "REFILL_REQUEST"
	DQEventTypeTransmissionError             = "TRANSMISSION_ERROR"
	DQEventTypeUnlinkedDNTFTransmissionError = "UNLINKED_DNTF_TRANSMISSION_ERROR"
	DQEventTypeRefillTransmissionError       = "REFILL_TRANSMISSION_ERROR"
	DQEventTypeCaseMessage                   = "CASE_MESSAGE"
	DQEventTypeCaseAssignment                = "CASE_ASSIGNMENT"
	DQItemStatusPending                      = "PENDING"
	DQItemStatusTreated                      = "TREATED"
	DQItemStatusTriaged                      = "TRIAGED"
	DQItemStatusOngoing                      = "ONGOING"
	DQItemStatusRefillApproved               = "APPROVED"
	DQItemStatusRefillDenied                 = "DENIED"
	DQItemStatusReplied                      = "REPLIED"
	DQItemStatusRead                         = "READ"
)

type DoctorQueueItem struct {
	ID                   int64
	DoctorID             int64
	DoctorContextID      int64 // id of the doctor/ma requesting the information
	EventType            string
	EnqueueDate          time.Time
	CompletedDate        time.Time
	Expires              *time.Time
	ItemID               int64
	Status               string
	PatientCaseID        int64
	PositionInQueue      int
	CareProvidingStateID int64
}

type ByTimestamp []*DoctorQueueItem

func (a ByTimestamp) Len() int      { return len(a) }
func (a ByTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool {
	return a[i].EnqueueDate.Before(a[j].EnqueueDate)
}

func (d *DoctorQueueItem) GetID() int64 {
	return d.ID
}

func (d *DoctorQueueItem) GetTitleAndSubtitle(dataAPI DataAPI) (string, string, error) {
	var title, subtitle string

	switch d.EventType {
	case DQEventTypePatientVisit, DQEventTypeTreatmentPlan:
		var patient *common.Patient
		var err error

		if d.EventType == DQEventTypeTreatmentPlan {
			patient, err = dataAPI.GetPatientFromTreatmentPlanID(d.ItemID)
			if err == NoRowsError {
				golog.Errorf("Did not get patient from treatment plan id (%d)", d.ItemID)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
		} else {
			patient, err = dataAPI.GetPatientFromPatientVisitID(d.ItemID)
			if err == NoRowsError {
				golog.Errorf("Did not get patient from patient visit id (%d)", d.ItemID)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
		}

		switch d.Status {
		case DQItemStatusTreated:
			title = fmt.Sprintf("Treatment Plan completed for %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusPending:
			title = fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusOngoing:
			title = fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusTriaged:
			title = fmt.Sprintf("Completed and triaged visit for %s %s", patient.FirstName, patient.LastName)
		}

	case DQEventTypeRefillRequest:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from refill request id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case DQItemStatusPending:
			title = fmt.Sprintf("Refill request for %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusRefillApproved:
			title = fmt.Sprintf("Refill request approved for %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusRefillDenied:
			title = fmt.Sprintf("Refill request denied for %s %s", patient.FirstName, patient.LastName)
		}

	case DQEventTypeRefillTransmissionError:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from refill request: %s", err)
			return "", "", err
		}

		switch d.Status {
		case DQItemStatusPending:
			title = fmt.Sprintf("Error completing refill request for %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusTreated:
			title = fmt.Sprintf("Refill request error resolved for %s %s", patient.FirstName, patient.LastName)
		}

	case DQEventTypeTransmissionError:
		patient, err := dataAPI.GetPatientFromTreatmentID(d.ItemID)
		if err == NoRowsError {
			golog.Errorf("Unable to get patient from treatment id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from treatment id %s", err)
			return "", "", err
		}

		switch d.Status {
		case DQItemStatusPending, DQItemStatusOngoing:
			title = fmt.Sprintf("Error sending prescription for %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusTreated:
			title = fmt.Sprintf("Error resolved for %s %s", patient.FirstName, patient.LastName)
		}

	case DQEventTypeUnlinkedDNTFTransmissionError:
		unlinkedTreatment, err := dataAPI.GetUnlinkedDNTFTreatment(d.ItemID)
		if err == NoRowsError {
			golog.Errorf("Unable to get unlinked dntf treatment from id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get unlinked dntf treatment from id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case DQItemStatusPending, DQItemStatusOngoing:
			title = fmt.Sprintf("Error sending prescription for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
		case DQItemStatusTreated:
			title = fmt.Sprintf("Error resolved for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
		}
	case DQEventTypeCaseMessage:

		patient, err := dataAPI.GetPatientFromCaseID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from case id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case DQItemStatusPending:
			title = fmt.Sprintf("Message from %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusRead:
			title = fmt.Sprintf("Conversation with %s %s", patient.FirstName, patient.LastName)
		case DQItemStatusReplied:
			title = fmt.Sprintf("Replied to %s %s", patient.FirstName, patient.LastName)
		}
	case DQEventTypeCaseAssignment:

		patient, err := dataAPI.GetPatientFromCaseID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from case id: %s", err)
			return "", "", err
		}

		assignments, err := dataAPI.GetActiveMembersOfCareTeamForCase(d.ItemID, true)
		if err != nil {
			golog.Errorf("Unable to get active members of care team for case: %s", err)
			return "", "", err
		}

		// determine the long display name of the other provider
		var otherProviderLongDisplayName, selfLongDisplayName string
		for _, assignment := range assignments {
			if assignment.ProviderID != d.DoctorContextID {
				otherProviderLongDisplayName = assignment.LongDisplayName
			} else {
				selfLongDisplayName = assignment.LongDisplayName
			}
		}

		switch d.Status {
		case DQItemStatusPending:
			caseAssignee := "you"
			if d.DoctorContextID != d.DoctorID {
				caseAssignee = otherProviderLongDisplayName
			}
			title = fmt.Sprintf("%s %s's case assigned to %s", patient.FirstName, patient.LastName, caseAssignee)
		case DQItemStatusReplied:
			caseAssignee := otherProviderLongDisplayName
			if d.DoctorContextID != d.DoctorID {
				caseAssignee = selfLongDisplayName
			}
			title = fmt.Sprintf("%s %s's case assigned to %s", patient.FirstName, patient.LastName, caseAssignee)
		}
	}
	return title, subtitle, nil
}

func (d *DoctorQueueItem) GetImageURL() *app_url.SpruceAsset {
	switch d.EventType {
	case DQEventTypePatientVisit:
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
	return []string{DisplayTypeTitleSubtitleActionable}
}

func (d *DoctorQueueItem) ActionURL(dataAPI DataAPI) (*app_url.SpruceAction, error) {
	switch d.EventType {
	case DQEventTypePatientVisit:
		patientVisit, err := dataAPI.GetPatientVisitFromID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient visit based on id: %s", err)
			return nil, err
		}

		switch d.Status {
		case DQItemStatusOngoing, DQItemStatusPending, DQItemStatusTriaged:
			return app_url.ViewPatientVisitInfoAction(patientVisit.PatientID.Int64(), d.ItemID, patientVisit.PatientCaseID.Int64()), nil
		}
	case DQEventTypeTreatmentPlan:
		treatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(d.ItemID, d.DoctorID)
		if err != nil {
			golog.Errorf("Unable to get treatment plan from id: %s", err)
			return nil, err
		}

		switch d.Status {
		case DQItemStatusTreated, DQItemStatusTriaged:
			return app_url.ViewCompletedTreatmentPlanAction(treatmentPlan.PatientID, d.ItemID, treatmentPlan.PatientCaseID.Int64()), nil
		}
	case DQEventTypeRefillTransmissionError:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id: %s", err)
			return nil, nil
		}

		return app_url.ViewRefillRequestAction(patient.PatientID.Int64(), d.ItemID), nil
	case DQEventTypeRefillRequest:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return nil, nil
		}

		switch d.Status {
		case DQItemStatusOngoing, DQItemStatusPending:
			return app_url.ViewRefillRequestAction(patient.PatientID.Int64(), d.ItemID), nil
		case DQItemStatusTreated, DQItemStatusRefillApproved, DQItemStatusRefillDenied:
			return app_url.ViewPatientTreatmentsAction(patient.PatientID.Int64()), nil
		}
	case DQEventTypeTransmissionError:
		patient, err := dataAPI.GetPatientFromTreatmentID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from treatment id : %s", err)
			return nil, nil
		}
		return app_url.ViewTransmissionErrorAction(patient.PatientID.Int64(), d.ItemID), nil
	case DQEventTypeUnlinkedDNTFTransmissionError:
		patient, err := dataAPI.GetPatientFromUnlinkedDNTFTreatment(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from unlinked dntf treatment: %s", err)
			return nil, nil
		}
		return app_url.ViewDNTFTransmissionErrorAction(patient.PatientID.Int64(), d.ItemID), nil
	case DQEventTypeCaseMessage, DQEventTypeCaseAssignment:

		// better to get the patient case object instead of the patient object here
		// because it lesser queries are made to get to the same information
		patientCase, err := dataAPI.GetPatientCaseFromID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from case id: %s", err)
			return nil, err
		}

		return app_url.ViewPatientMessagesAction(patientCase.PatientID.Int64(), d.ItemID), nil
	}

	return nil, nil
}
