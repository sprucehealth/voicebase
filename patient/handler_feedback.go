package patient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
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
	dataAPI       api.DataAPI
	taggingClient tagging.Client
	cfgStore      cfg.Store
}

type feedbackSubmitRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment,omitempty"`
}

// lowRatingTagThreshold is a Server configurable value for the threshold at which to tag the patient's case as LowRating
var lowRatingTagThreshold = &cfg.ValueDef{
	Name:        "Patient.Feedback.LowRating.Tag.Threshold",
	Description: "A value that represents the threshold for which if a patient feedback rating is equal to or below, the latest case for the patient will be tagged as LowRating.",
	Type:        cfg.ValueTypeInt,
	Default:     3,
}

const (
	LowRatingTag string = "LowRating"
)

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
		httputil.Get)
}

func NewFeedbackHandler(dataAPI api.DataAPI, taggingClient tagging.Client, cfgStore cfg.Store) http.Handler {
	cfgStore.Register(lowRatingTagThreshold)
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&feedbackHandler{
				dataAPI:       dataAPI,
				taggingClient: taggingClient,
				cfgStore:      cfgStore,
			}),
			[]string{api.RolePatient}),
		httputil.Post)
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

	// Check to see if we need to tag the latest case for a low rating but don't block/fail the API on this.
	go func() {
		lowRatingThreshold := h.cfgStore.Snapshot().Int(lowRatingTagThreshold.Name)
		if req.Rating <= lowRatingThreshold {
			if _, err = h.taggingClient.InsertTagAssociation(&model.Tag{Text: LowRatingTag}, &model.TagMembership{
				CaseID: ptr.Int64Ptr(tp.PatientCaseID.Int64()),
				// Place this tag in immediate trigger violation
				TriggerTime: ptr.TimePtr(time.Now()),
				Hidden:      false,
			}); err != nil {
				golog.Errorf("%v", err)
			}
		}
	}()

	apiservice.WriteJSONSuccess(w)
}
