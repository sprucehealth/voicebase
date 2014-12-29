package messages

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

const AttachmentTypePrefix = "attachment:"

func validateAccess(dataAPI api.DataAPI, r *http.Request, patientCase *common.PatientCase) (personID, doctorID int64, err error) {
	ctx := apiservice.GetContext(r)
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		doctorID, err = dataAPI.GetDoctorIDFromAccountID(ctx.AccountID)
		if err != nil {
			return 0, 0, err
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctx.Role, doctorID, patientCase.PatientID.Int64(), patientCase.ID.Int64(), dataAPI); err != nil {
			return 0, 0, err
		}

		personID, err = dataAPI.GetPersonIDByRole(api.DOCTOR_ROLE, doctorID)
		if err != nil {
			return 0, 0, err
		}
	case api.PATIENT_ROLE:
		patientID, err := dataAPI.GetPatientIDFromAccountID(ctx.AccountID)
		if err != nil {
			return 0, 0, err
		}
		if patientCase.PatientID.Int64() != patientID {
			return 0, 0, apiservice.NewValidationError("Not authorized", r)
		}
		personID, err = dataAPI.GetPersonIDByRole(api.PATIENT_ROLE, patientID)
		if err != nil {
			return 0, 0, err
		}
	case api.MA_ROLE:
		// For messaging, we let the MA POST as well as GET from the message thread given
		// they will be an active participant in the thread.
		doctorID, err = dataAPI.GetDoctorIDFromAccountID(ctx.AccountID)
		if err != nil {
			return 0, 0, err
		}

		personID, err = dataAPI.GetPersonIDByRole(api.MA_ROLE, doctorID)
		if err != nil {
			return 0, 0, err
		}

	default:
		return 0, 0, errors.New("Unknown role " + ctx.Role)
	}

	return personID, doctorID, nil
}

func CreateMessageAndAttachments(msg *common.CaseMessage, attachments []*Attachment, personID, doctorID int64, role string, dataAPI api.DataAPI) error {
	// Validate all attachments
	for _, att := range attachments {
		switch att.Type {
		default:
			return apiservice.NewError("Unknown attachment type "+att.Type, http.StatusBadRequest)
		case common.AttachmentTypeTreatmentPlan:
			// Make sure the treatment plan is a part of the same case
			if role != api.DOCTOR_ROLE {
				return apiservice.NewError("Only a doctor is allowed to attach a treatment plan", http.StatusBadRequest)
			}
			tp, err := dataAPI.GetAbridgedTreatmentPlan(att.ID, doctorID)
			if err != nil {
				return err
			}
			if tp.PatientCaseID.Int64() != msg.CaseID {
				return apiservice.NewError("Treatment plan does not belong to the case", http.StatusBadRequest)
			}
			if tp.DoctorID.Int64() != doctorID {
				return apiservice.NewError("Treatment plan not created by the requesting doctor", http.StatusBadRequest)
			}
		case common.AttachmentTypeVisit:
			// Make sure the visit is part of the same case
			if role != api.DOCTOR_ROLE && role != api.MA_ROLE {
				return apiservice.NewError("Only a doctor is allowed to attach a visit", http.StatusBadRequest)
			}
			visit, err := dataAPI.GetPatientVisitFromID(att.ID)
			if err != nil {
				return err
			}
			if visit.PatientCaseID.Int64() != msg.CaseID {
				return apiservice.NewError("visit does not belong to the case", http.StatusBadRequest)
			}
		case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
			// Make sure media is uploaded by the same person
			media, err := dataAPI.GetMedia(att.ID)
			if err != nil {
				return err
			}
			if media.UploaderID != personID {
				return apiservice.NewError("Invalid attachment", http.StatusBadRequest)
			}
		}
		title := att.Title
		if title == "" {
			title = AttachmentTitle(att.Type)
		}
		msg.Attachments = append(msg.Attachments, &common.CaseMessageAttachment{
			ItemType: att.Type,
			ItemID:   att.ID,
			Title:    title,
		})
	}

	msgID, err := dataAPI.CreateCaseMessage(msg)
	if err != nil {
		return err
	}

	msg.ID = msgID
	return nil

}
