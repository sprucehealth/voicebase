package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/compat"
	"github.com/sprucehealth/backend/cmd/svc/restapi/internal/httputil"
)

type patientCapabilitiesHandler struct {
	dataAPI  api.DataAPI
	authAPI  api.AuthAPI
	features compat.Features
}

type patientCapabilitiesRequest struct {
	PatientID common.PatientID `json:"patient_id" schema:"patient_id,schema"`
}

type patientCapabilitiesResponse struct {
	Features []string `json:"features"`
}

// NewPatientCapabilitiesHandler returns a new handler that returns patient compatiblity flags.
func NewPatientCapabilitiesHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, features compat.Features) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&patientCapabilitiesHandler{
				dataAPI:  dataAPI,
				authAPI:  authAPI,
				features: features,
			}),
			api.RoleCC, api.RoleDoctor),
		httputil.Get)
}

func (h *patientCapabilitiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req patientCapabilitiesRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}
	p, err := h.dataAPI.Patient(req.PatientID, true)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError("patient not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	appInfo, err := h.authAPI.LatestAppInfo(p.AccountID.Int64())
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError("app info not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &patientCapabilitiesResponse{
		Features: h.features.Set(appInfo.Platform.String()+"-patient", appInfo.Version).Enumerate(),
	})
}
