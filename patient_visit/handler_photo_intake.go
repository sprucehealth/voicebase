package patient_visit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/httputil"
)

type photoAnswerIntakeHandler struct {
	dataAPI api.DataAPI
}

type PhotoAnswerIntakeResponse struct {
	Result string `json:"result"`
}

type PhotoAnswerIntakeQuestionItem struct {
	QuestionID    int64                        `json:"question_id,string"`
	PhotoSections []*common.PhotoIntakeSection `json:"answered_photo_sections"`
}

type PhotoAnswerIntakeRequestData struct {
	PhotoQuestions []*PhotoAnswerIntakeQuestionItem `json:"photo_questions"`
	SessionID      string                           `json:"session_id"`
	SessionCounter uint                             `json:"counter"`
	PatientVisitID int64                            `json:"patient_visit_id,string"`
}

func (r *PhotoAnswerIntakeRequestData) Validate() (bool, string) {
	// TODO: the validation isn't comprehensive as I'm not sure right now what all to check.
	if r.PatientVisitID <= 0 {
		return false, "patient visit ID is required"
	}
	for _, pq := range r.PhotoQuestions {
		if pq.QuestionID <= 0 {
			return false, "question ID is required"
		}
		for _, ps := range pq.PhotoSections {
			for _, p := range ps.Photos {
				if p.PhotoID <= 0 {
					return false, "photo ID is required"
				}
			}
		}
	}
	return true, ""
}

func NewPhotoAnswerIntakeHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(apiservice.AuthorizationRequired(
		&photoAnswerIntakeHandler{
			dataAPI: dataAPI,
		}), httputil.Post)
}

func (p *photoAnswerIntakeHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.RolePatient {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (p *photoAnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData PhotoAnswerIntakeRequestData
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	if valid, reason := requestData.Validate(); !valid {
		// FIXME: logging this for now as we've been seeing likely bad requests recently. can remove
		//        after no longer needed for debug.
		golog.Warningf("invalid request to photo answer intake: %s", reason)
		apiservice.WriteValidationError(reason, w, r)
		return
	}

	patientID, err := p.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientIDFromPatientVisitID, err := p.dataAPI.GetPatientIDFromPatientVisitID(requestData.PatientVisitID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if patientIDFromPatientVisitID != patientID {
		apiservice.WriteValidationError("patient id retrieved from the patient_visit_id does not match patient id retrieved from auth token", w, r)
		return
	}

	for _, photoIntake := range requestData.PhotoQuestions {
		// ensure that intake is for the right question type
		questionType, err := p.dataAPI.GetQuestionType(photoIntake.QuestionID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if questionType != info_intake.QuestionTypePhotoSection {
			apiservice.WriteValidationError("only photo section question types acceptable for intake via this endpoint", w, r)
			return
		}

		// get photo slots for the question and ensure that all slot ids in the request
		// belong to this question
		photoSlots, err := p.dataAPI.GetPhotoSlotsInfo(photoIntake.QuestionID, api.LanguageIDEnglish)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		photoSlotIDMapping := make(map[int64]bool, len(photoSlots))
		for _, photoSlot := range photoSlots {
			photoSlotIDMapping[photoSlot.ID] = true
		}

		for _, photoSection := range photoIntake.PhotoSections {
			for _, photo := range photoSection.Photos {
				if !photoSlotIDMapping[photo.SlotID] {
					apiservice.WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("Slot id %d not associated with photo question id %d: ", photo.SlotID, photoIntake.QuestionID))
					return
				}
			}
		}

		if err := p.dataAPI.StorePhotoSectionsForQuestion(
			photoIntake.QuestionID, patientID, requestData.PatientVisitID,
			requestData.SessionID,
			requestData.SessionCounter,
			photoIntake.PhotoSections,
		); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}
