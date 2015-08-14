package patient_case

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type patientFeedbackHandler struct {
	dataAPI api.DataAPI
}

type patientFeedbackResponse struct {
	Feedback []*patientFeedback `json:"feedback"`
}

type patientFeedback struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
	Created int64  `json:"created_timestamp"`
}

func NewPatientFeedbackHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&patientFeedbackHandler{
				dataAPI: dataAPI,
			}), api.RoleCC), httputil.Get)
}

func (h *patientFeedbackHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		apiservice.WriteBadRequestError(ctx, errors.New("case_id required"), w, r)
		return
	}
	feedback, err := h.dataAPI.PatientFeedback("case:" + strconv.FormatInt(caseID, 10))
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	res := &patientFeedbackResponse{
		Feedback: make([]*patientFeedback, len(feedback)),
	}
	for i, f := range feedback {
		res.Feedback[i] = &patientFeedback{
			Rating:  f.Rating,
			Comment: f.Comment,
			Created: f.Created.Unix(),
		}
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}
