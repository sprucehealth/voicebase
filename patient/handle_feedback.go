package patient

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

const (
	textTagFeedbackScreenTitle        = "txt_feedback_screen_title"
	textTagFeedbackRatingPrompt       = "txt_feedback_rating_prompt"
	textTagFeedbackCommentPlaceholder = "txt_feedback_comment_placeholder"
	textTagFeedbackSubmitButton       = "txt_feedback_submit_button"
)

var feedbackTextTags = []string{
	textTagFeedbackScreenTitle,
	textTagFeedbackRatingPrompt,
	textTagFeedbackCommentPlaceholder,
	textTagFeedbackSubmitButton,
}

type feedbackPromptHandler struct {
	dataAPI api.DataAPI
}

type feedbackHandler struct {
	dataAPI api.DataAPI
}

type feedbackSubmitRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment,omitempty"`
}

type feedbackPromptResponse struct {
	ScreenTitle        string `json:"screen_title"`
	RatingPromptText   string `json:"rating_prompt_text"`
	CommentPlaceholder string `json:"comment_placeholder"`
	SubmitButtonText   string `json:"submit_button_text"`
}

func NewFeedbackPromptHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&feedbackPromptHandler{
				dataAPI: dataAPI,
			}),
			[]string{api.RolePatient}),
		[]string{httputil.Get})
}

func NewFeedbackHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&feedbackHandler{
				dataAPI: dataAPI,
			}),
			[]string{api.RolePatient}),
		[]string{httputil.Post})
}

func (h *feedbackPromptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	text, err := h.dataAPI.LocalizedText(api.LanguageIDEnglish, feedbackTextTags)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	res := &feedbackPromptResponse{
		ScreenTitle:        text[textTagFeedbackScreenTitle],
		RatingPromptText:   text[textTagFeedbackRatingPrompt],
		CommentPlaceholder: text[textTagFeedbackCommentPlaceholder],
		SubmitButtonText:   text[textTagFeedbackSubmitButton],
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (h *feedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req feedbackSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}
	patientID, err := h.dataAPI.GetPatientIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	tp, err := latestActiveTreatmentPlan(h.dataAPI, patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if tp == nil {
		golog.Errorf("Feedback submitted with no active treatment plan for patient %d", patientID)
		apiservice.WriteJSONSuccess(w)
		return
	}
	if err := h.dataAPI.RecordPatientFeedback(patientID, fmt.Sprintf("case:%d", tp.PatientCaseID.Int64()), req.Rating, req.Comment); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}
