package main

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

func getPatientID(dataAPI api.DataAPI, d *api.DoctorQueueItem) (int64, error) {
	var patient *common.Patient
	var err error

	switch d.EventType {
	case api.DQEventTypePatientVisit:
		patient, err = dataAPI.GetPatientFromPatientVisitID(d.ItemID)
		if err != nil {
			return 0, err
		}
	case api.DQEventTypeTreatmentPlan:
		patient, err = dataAPI.GetPatientFromTreatmentPlanID(d.ItemID)
		if err != nil {
			return 0, err
		}
	case api.DQEventTypeRefillRequest, api.DQEventTypeRefillTransmissionError:
		patient, err = dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err != nil {
			return 0, err
		}
	case api.DQEventTypeTransmissionError:
		patient, err = dataAPI.GetPatientFromTreatmentID(d.ItemID)
		if err != nil {
			return 0, err
		}
	case api.DQEventTypeUnlinkedDNTFTransmissionError:
		unlinkedTreatment, err := dataAPI.GetUnlinkedDNTFTreatment(d.ItemID)
		if err != nil {
			return 0, err
		}
		patient = unlinkedTreatment.Patient
	case api.DQEventTypeCaseMessage, api.DQEventTypeCaseAssignment:
		patient, err = dataAPI.GetPatientFromCaseID(d.ItemID)
		if err != nil {
			return 0, err
		}
	}

	return patient.PatientID.Int64(), nil
}

func getLongAndShortDescription(dataAPI api.DataAPI, d *api.DoctorQueueItem) (string, string, error) {
	var description, shortDescription string

	doctor, err := dataAPI.Doctor(d.DoctorID, true)
	if err != nil {
		return "", "", err
	}

	switch d.EventType {
	case api.DQEventTypePatientVisit, api.DQEventTypeTreatmentPlan:
		var patient *common.Patient
		var err error

		if d.EventType == api.DQEventTypeTreatmentPlan {
			patient, err = dataAPI.GetPatientFromTreatmentPlanID(d.ItemID)
			if err == api.NoRowsError {
				golog.Errorf("Did not get patient from treatment plan id (%d)", d.ItemID)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
		} else {
			patient, err = dataAPI.GetPatientFromPatientVisitID(d.ItemID)
			if err == api.NoRowsError {
				golog.Errorf("Did not get patient from patient visit id (%d)", d.ItemID)
				return "", "", nil
			} else if err != nil {
				return "", "", err
			}
		}

		switch d.Status {
		case api.DQItemStatusTreated:
			description = fmt.Sprintf("%s completed treatment plan for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Treatment plan completed by %s", doctor.ShortDisplayName)
		case api.DQItemStatusPending:
			description = fmt.Sprintf("New visit with %s %s", patient.FirstName, patient.LastName)
			shortDescription = "New visit"
		case api.DQItemStatusOngoing:
			description = fmt.Sprintf("Continue reviewing visit with %s %s", patient.FirstName, patient.LastName)
			shortDescription = "New visit"
		case api.DQItemStatusTriaged:
			description = fmt.Sprintf("%s completed and triaged visit for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Visit triaged by %s", doctor.ShortDisplayName)
		}

	case api.DQEventTypeRefillRequest:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err == api.NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from refill request id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending:
			description = fmt.Sprintf("Refill request for %s %s", patient.FirstName, patient.LastName)
			shortDescription = "New refill request"
		case api.DQItemStatusRefillApproved:
			description = fmt.Sprintf("%s approved refill request for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Refill request approved by %s", doctor.ShortDisplayName)
		case api.DQItemStatusRefillDenied:
			description = fmt.Sprintf("%s denied refill request for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Refill request denied by %s", doctor.ShortDisplayName)
		}

	case api.DQEventTypeRefillTransmissionError:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err == api.NoRowsError {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from refill request: %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending:
			description = fmt.Sprintf("Error completing refill request for %s %s", patient.FirstName, patient.LastName)
			shortDescription = "Refill request error"
		case api.DQItemStatusTreated:
			description = fmt.Sprintf("%s resolved refill request error for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Refill request error resolved by %s", doctor.ShortDisplayName)
		}

	case api.DQEventTypeTransmissionError:
		patient, err := dataAPI.GetPatientFromTreatmentID(d.ItemID)
		if err == api.NoRowsError {
			golog.Errorf("Unable to get patient from treatment id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get patient from treatment id %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending, api.DQItemStatusOngoing:
			description = fmt.Sprintf("Error sending prescription for %s %s", patient.FirstName, patient.LastName)
			shortDescription = "Prescription error"
		case api.DQItemStatusTreated:
			description = fmt.Sprintf("%s resolved prescription error for %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Prescription error resolved by %s", doctor.ShortDisplayName)
		}

	case api.DQEventTypeUnlinkedDNTFTransmissionError:
		unlinkedTreatment, err := dataAPI.GetUnlinkedDNTFTreatment(d.ItemID)
		if err == api.NoRowsError {
			golog.Errorf("Unable to get unlinked dntf treatment from id %d", d.ItemID)
			return "", "", nil
		} else if err != nil {
			golog.Errorf("Unable to get unlinked dntf treatment from id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending, api.DQItemStatusOngoing:
			description = fmt.Sprintf("Error sending prescription for %s %s", unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
			shortDescription = "Prescription error"
		case api.DQItemStatusTreated:
			description = fmt.Sprintf("%s resolved prescription error for %s %s", doctor.ShortDisplayName, unlinkedTreatment.Patient.FirstName, unlinkedTreatment.Patient.LastName)
			shortDescription = fmt.Sprintf("Prescription error resolved by %s", doctor.ShortDisplayName)
		}
	case api.DQEventTypeCaseMessage:

		patient, err := dataAPI.GetPatientFromCaseID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from case id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending:
			description = fmt.Sprintf("Message from %s %s", patient.FirstName, patient.LastName)
			shortDescription = "New message"
		case api.DQItemStatusRead:
			description = fmt.Sprintf("Conversation with %s %s", patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Conversation revewed by %s", doctor.ShortDisplayName)
		case api.DQItemStatusReplied:
			description = fmt.Sprintf("%s replied to %s %s", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Messaged by %s", doctor.ShortDisplayName)
		}
	case api.DQEventTypeCaseAssignment:

		patient, err := dataAPI.GetPatientFromCaseID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from case id: %s", err)
			return "", "", err
		}

		switch d.Status {
		case api.DQItemStatusPending:
			careTeamMembers, err := dataAPI.GetActiveMembersOfCareTeamForCase(d.ItemID, true)
			if err != nil {
				golog.Errorf("Unable to get members of care team: %s", err)
				return "", "", err
			}

			var otherProviderShortDisplayName string
			for _, member := range careTeamMembers {
				if member.ProviderID != doctor.DoctorID.Int64() {
					otherProviderShortDisplayName = member.ShortDisplayName
					break
				}
			}
			description = fmt.Sprintf("%s %s's case assigned to %s", patient.FirstName, patient.LastName, doctor.ShortDisplayName)
			shortDescription = fmt.Sprintf("Reassigned by %s", otherProviderShortDisplayName)
		case api.DQItemStatusReplied:
			description = fmt.Sprintf("%s assigned %s %s's case", doctor.ShortDisplayName, patient.FirstName, patient.LastName)
			shortDescription = fmt.Sprintf("Assigned to %s", doctor.ShortDisplayName)
		}
	}
	return description, shortDescription, nil
}

func getActionURL(dataAPI api.DataAPI, d *api.DoctorQueueItem) (*app_url.SpruceAction, error) {
	switch d.EventType {
	case api.DQEventTypePatientVisit:
		patientVisit, err := dataAPI.GetPatientVisitFromID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient visit based on id: %s", err)
			return nil, err
		}

		switch d.Status {
		case api.DQItemStatusOngoing, api.DQItemStatusPending, api.DQItemStatusTriaged:
			return app_url.ViewPatientVisitInfoAction(patientVisit.PatientID.Int64(), d.ItemID, patientVisit.PatientCaseID.Int64()), nil
		}
	case api.DQEventTypeTreatmentPlan:
		treatmentPlan, err := dataAPI.GetAbridgedTreatmentPlan(d.ItemID, d.DoctorID)
		if err != nil {
			golog.Errorf("Unable to get treatment plan from id: %s", err)
			return nil, err
		}

		switch d.Status {
		case api.DQItemStatusTreated, api.DQItemStatusTriaged:
			return app_url.ViewCompletedTreatmentPlanAction(treatmentPlan.PatientID, d.ItemID, treatmentPlan.PatientCaseID.Int64()), nil
		}
	case api.DQEventTypeRefillTransmissionError:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id: %s", err)
			return nil, nil
		}

		return app_url.ViewRefillRequestAction(patient.PatientID.Int64(), d.ItemID), nil
	case api.DQEventTypeRefillRequest:
		patient, err := dataAPI.GetPatientFromRefillRequestID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from refill request id %d", d.ItemID)
			return nil, nil
		}

		switch d.Status {
		case api.DQItemStatusOngoing, api.DQItemStatusPending:
			return app_url.ViewRefillRequestAction(patient.PatientID.Int64(), d.ItemID), nil
		case api.DQItemStatusTreated, api.DQItemStatusRefillApproved, api.DQItemStatusRefillDenied:
			return app_url.ViewPatientTreatmentsAction(patient.PatientID.Int64()), nil
		}
	case api.DQEventTypeTransmissionError:
		patient, err := dataAPI.GetPatientFromTreatmentID(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from treatment id : %s", err)
			return nil, nil
		}
		return app_url.ViewTransmissionErrorAction(patient.PatientID.Int64(), d.ItemID), nil
	case api.DQEventTypeUnlinkedDNTFTransmissionError:
		patient, err := dataAPI.GetPatientFromUnlinkedDNTFTreatment(d.ItemID)
		if err != nil {
			golog.Errorf("Unable to get patient from unlinked dntf treatment: %s", err)
			return nil, nil
		}
		return app_url.ViewDNTFTransmissionErrorAction(patient.PatientID.Int64(), d.ItemID), nil
	case api.DQEventTypeCaseMessage, api.DQEventTypeCaseAssignment:

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
