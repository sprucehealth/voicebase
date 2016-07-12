package patient

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/feedback"
	"github.com/sprucehealth/backend/cmd/svc/restapi/tagging"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
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
	dataAPI        api.DataAPI
	feedbackClient feedback.DAL
	taggingClient  tagging.Client
	cfgStore       cfg.Store
}

type additionalFeedback struct {
	TemplateID int64           `json:"id,string"`
	Answer     json.RawMessage `json:"answer"`
}

type feedbackSubmitRequest struct {
	Rating             int                 `json:"rating"`
	Comment            *string             `json:"comment,omitempty"`
	AdditionalFeedback *additionalFeedback `json:"additional_feedback"`
}

// lowRatingTagThreshold is a Server configurable value for the threshold at which to tag the patient's case to be marked for follow up
var lowRatingTagThreshold = &cfg.ValueDef{
	Name:        "Patient.Feedback.LowRating.Tag.Threshold",
	Description: "The threshold for which if a patient feedback rating is equal to or below, the latest case for the patient will be marked for follow up.",
	Type:        cfg.ValueTypeInt,
	Default:     3,
}

const (
	// LowRatingTag is the tag to be applied to feedback that matches the qualifications to be "low"
	LowRatingTag = "LowRating"
)

type feedbackPromptResponse struct {
	ScreenTitle        string `json:"screen_title"`
	RatingPromptText   string `json:"rating_prompt_text"`
	CommentPlaceholder string `json:"comment_placeholder"`
	SubmitButtonText   string `json:"submit_button_text"`
}

// NewFeedbackPromptHandler returns an initialized instance of feedbackPromptHandler
func NewFeedbackPromptHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&feedbackPromptHandler{
				dataAPI: dataAPI,
			}),
			api.RolePatient),
		httputil.Get)
}

// NewFeedbackHandler returns an initialized instance of feedbackHandler
func NewFeedbackHandler(dataAPI api.DataAPI, feedbackClient feedback.DAL, taggingClient tagging.Client, cfgStore cfg.Store) httputil.ContextHandler {
	cfgStore.Register(lowRatingTagThreshold)
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&feedbackHandler{
				dataAPI:        dataAPI,
				feedbackClient: feedbackClient,
				taggingClient:  taggingClient,
				cfgStore:       cfgStore,
			}),
			api.RolePatient),
		httputil.Post)
}

func (h *feedbackPromptHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	text, err := h.dataAPI.LocalizedText(api.LanguageIDEnglish, feedbackTextTags)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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

func (h *feedbackHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req feedbackSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiservice.WriteBadRequestError(ctx, err, w, r)
		return
	}
	patientID, err := h.dataAPI.GetPatientIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	tp, err := latestActiveTreatmentPlan(h.dataAPI, patientID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	if tp == nil {
		golog.Errorf("Feedback submitted with no active treatment plan for patient %s", patientID)
		apiservice.WriteJSONSuccess(w)
		return
	}

	var structuredResponse feedback.StructuredResponse
	if req.AdditionalFeedback != nil {
		feedbackTemplate, err := h.feedbackClient.FeedbackTemplate(req.AdditionalFeedback.TemplateID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		structuredResponse, err = feedbackTemplate.Template.ParseAndValidateResponse(feedbackTemplate.ID, []byte(req.AdditionalFeedback.Answer))
		if err != nil {
			apiservice.WriteValidationError(ctx, errors.Cause(err).Error(), w, r)
			return
		}
	}

	if err := h.feedbackClient.RecordPatientFeedback(
		patientID,
		feedback.ForCase(tp.PatientCaseID.Int64()),
		req.Rating,
		req.Comment,
		structuredResponse,
	); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	conc.Go(func() {
		lowRatingThreshold := h.cfgStore.Snapshot().Int(lowRatingTagThreshold.Name)
		if req.Rating <= lowRatingThreshold {
			if err := tagging.ApplyCaseTag(h.taggingClient, LowRatingTag, tp.PatientCaseID.Int64(), ptr.Time(time.Now()), tagging.TONone); err != nil {
				golog.Errorf("%v", err)
			}
		}
		if err := tagging.ApplyCaseTag(h.taggingClient, "rating:"+strconv.FormatInt(int64(req.Rating), 10), tp.PatientCaseID.Int64(), nil, tagging.TONone); err != nil {
			golog.Errorf("%v", err)
		}
	})

	apiservice.WriteJSONSuccess(w)
}
