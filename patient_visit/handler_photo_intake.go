package patient_visit

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/httputil"
)

type photoAnswerIntakeHandler struct {
	dataApi api.DataAPI
}

type PhotoAnswerIntakeResponse struct {
	Result string `json:"result"`
}

type PhotoAnswerIntakeQuestionItem struct {
	QuestionId    int64                        `json:"question_id,string"`
	PhotoSections []*common.PhotoIntakeSection `json:"answered_photo_sections"`
}

type PhotoAnswerIntakeRequestData struct {
	PhotoQuestions []*PhotoAnswerIntakeQuestionItem `json:"photo_questions"`
	SessionID      string                           `json:"session_id"`
	SessionCounter uint                             `json:"counter"`
	PatientVisitId int64                            `json:"patient_visit_id,string"`
}

func NewPhotoAnswerIntakeHandler(dataApi api.DataAPI) http.Handler {
	return httputil.SupportedMethods(apiservice.AuthorizationRequired(
		&photoAnswerIntakeHandler{
			dataApi: dataApi,
		}), []string{"POST"})
}

func (p *photoAnswerIntakeHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (p *photoAnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestData PhotoAnswerIntakeRequestData
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientId, err := p.dataApi.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientIdFromPatientVisitId, err := p.dataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if patientIdFromPatientVisitId != patientId {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "patient id retrieved from the patient_visit_id does not match patient id retrieved from auth token")
		return
	}

	for _, photoIntake := range requestData.PhotoQuestions {
		// ensure that intake is for the right question type
		questionType, err := p.dataApi.GetQuestionType(photoIntake.QuestionId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		} else if questionType != info_intake.QUESTION_TYPE_PHOTO_SECTION {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "only photo section question types acceptable for intake via this endpoint")
			return
		}

		// get photo slots for the question and ensure that all slot ids in the request
		// belong to this question
		photoSlots, err := p.dataApi.GetPhotoSlots(photoIntake.QuestionId, api.EN_LANGUAGE_ID)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}

		photoSlotIdMapping := make(map[int64]bool)
		for _, photoSlot := range photoSlots {
			photoSlotIdMapping[photoSlot.Id] = true
		}

		for _, photoSection := range photoIntake.PhotoSections {
			for _, photo := range photoSection.Photos {
				if !photoSlotIdMapping[photo.SlotID] {
					apiservice.WriteUserError(w, http.StatusBadRequest, fmt.Sprintf("Slot id %d not associated with photo question id %d: ", photo.SlotID, photoIntake.QuestionId))
					return
				}
			}
		}

		if err := p.dataApi.StorePhotoSectionsForQuestion(
			photoIntake.QuestionId, patientId, requestData.PatientVisitId,
			requestData.SessionID,
			requestData.SessionCounter,
			photoIntake.PhotoSections); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
