package patient_case

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
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

func NewPatientFeedbackHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&patientFeedbackHandler{
			dataAPI: dataAPI,
		}), httputil.Get)
}

func (h *patientFeedbackHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.RoleCC {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (h *patientFeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caseID, err := strconv.ParseInt(r.FormValue("case_id"), 10, 64)
	if err != nil {
		apiservice.WriteBadRequestError(errors.New("case_id required"), w, r)
		return
	}
	feedback, err := h.dataAPI.PatientFeedback("case:" + strconv.FormatInt(caseID, 10))
	if err != nil {
		apiservice.WriteError(err, w, r)
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
