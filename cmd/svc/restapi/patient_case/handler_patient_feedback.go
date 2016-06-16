package patient_case

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type patientFeedbackHandler struct {
	feedbackClient feedback.DAL
}

type patientFeedbackResponse struct {
	Feedback []*patientFeedback `json:"feedback"`
}

type patientFeedback struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
	Created int64  `json:"created_timestamp"`
}

// NewPatientFeedbackHandler returns a handler that exposes
// patient feedback for the care coordinator
func NewPatientFeedbackHandler(feedbackClient feedback.DAL) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&patientFeedbackHandler{
				feedbackClient: feedbackClient,
			}), api.RoleCC), httputil.Get)
}

func (h *patientFeedbackHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		apiservice.WriteBadRequestError(ctx, errors.New("case_id required"), w, r)
		return
	}
	pf, err := h.feedbackClient.PatientFeedback(feedback.ForCase(caseID))
	if errors.Cause(err) != feedback.ErrNoPatientFeedback && err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if pf == nil {
		// no feedback exists.
		httputil.JSONResponse(w, http.StatusOK, &patientFeedbackResponse{})
		return
	}

	feedbackTemplate, responseJSON, err := h.feedbackClient.AdditionalFeedback(pf.ID)
	if errors.Cause(err) != feedback.ErrNoAdditionalFeedback && err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	var comment string
	if pf.Comment != nil {
		comment = *pf.Comment
	} else if feedbackTemplate != nil && responseJSON != nil {
		comment, err = feedbackTemplate.Template.ResponseString(feedbackTemplate.ID, responseJSON)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	var res patientFeedbackResponse
	if !pf.Pending && !pf.Dismissed {
		res.Feedback = []*patientFeedback{
			{
				Rating:  *pf.Rating,
				Comment: comment,
				Created: pf.Created.Unix(),
			},
		}
	}

	httputil.JSONResponse(w, http.StatusOK, res)
}
