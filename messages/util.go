package messages

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

func validateAccess(dataAPI api.DataAPI, r *http.Request, patientCase *common.PatientCase) (personID, doctorID int64, err error) {
	ctx := apiservice.GetContext(r)
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		doctorID, err = dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			return 0, 0, err
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctx.Role, doctorID, patientCase.PatientId.Int64(), patientCase.Id.Int64(), dataAPI); err != nil {
			return 0, 0, err
		}

		personID, err = dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorID)
		if err != nil {
			return 0, 0, err
		}
	case api.PATIENT_ROLE:
		patientID, err := dataAPI.GetPatientIdFromAccountId(ctx.AccountId)
		if err != nil {
			return 0, 0, err
		}
		if patientCase.PatientId.Int64() != patientID {
			return 0, 0, apiservice.NewValidationError("Not authorized", r)
		}
		personID, err = dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientID)
		if err != nil {
			return 0, 0, err
		}
	case api.MA_ROLE:
		// For messaging, we let the MA POST as well as GET from the message thread given
		// they will be an active participant in the thread.
		doctorID, err = dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			return 0, 0, err
		}

		personID, err = dataAPI.GetPersonIdByRole(api.MA_ROLE, doctorID)
		if err != nil {
			return 0, 0, err
		}

	default:
		return 0, 0, errors.New("Unknown role " + ctx.Role)
	}

	return personID, doctorID, nil
}

func createMessageAndAttachments(msg *common.CaseMessage, attachments []*Attachment, personID, doctorID int64, dataAPI api.DataAPI, r *http.Request) error {

	if attachments != nil {
		// Validate all attachments
		for _, att := range attachments {
			switch att.Type {
			default:
				return apiservice.NewValidationError("Unknown attachment type "+att.Type, r)
			case common.AttachmentTypeTreatmentPlan:
				// Make sure the treatment plan is a part of the same case
				if apiservice.GetContext(r).Role != api.DOCTOR_ROLE {
					return apiservice.NewValidationError("Only a doctor is allowed to attac a treatment plan", r)
				}
				tp, err := dataAPI.GetAbridgedTreatmentPlan(att.ID, doctorID)
				if err != nil {
					return err
				}
				if tp.PatientCaseId.Int64() != msg.CaseID {
					return apiservice.NewValidationError("Treatment plan does not belong to the case", r)
				}
				if tp.DoctorId.Int64() != doctorID {
					return apiservice.NewValidationError("Treatment plan not created by the requesting doctor", r)
				}
			case common.AttachmentTypePhoto:
				// Make sure the photo is uploaded by the same person and is unclaimed
				photo, err := dataAPI.GetPhoto(att.ID)
				if err != nil {
					return err
				}
				if photo.UploaderId != personID || photo.ClaimerType != "" {
					return apiservice.NewValidationError("Invalid attachment", r)
				}
			}
			msg.Attachments = append(msg.Attachments, &common.CaseMessageAttachment{
				ItemType: att.Type,
				ItemID:   att.ID,
			})
		}
	}

	msgID, err := dataAPI.CreateCaseMessage(msg)
	if err != nil {
		return err
	}

	msg.ID = msgID
	return nil

}
