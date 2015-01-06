package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type listHandler struct {
	dataAPI api.DataAPI
}

type listCasesRequestData struct {
	PatientID int64 `schema:"patient_id"`
}

type listCasesResponseData struct {
	PatientCases []*common.PatientCase `json:"cases"`
}

func NewListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&listHandler{
					dataAPI: dataAPI,
				}), []string{api.PATIENT_ROLE, api.MA_ROLE}),
		[]string{"GET"})
}

func (l *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	switch ctxt.Role {
	case api.PATIENT_ROLE:
		patientID, err := l.dataAPI.GetPatientIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientID] = patientID

	case api.DOCTOR_ROLE:
		var requestData listCasesRequestData
		if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
			return false, apiservice.NewValidationError(err.Error())
		}
		patientID := requestData.PatientID
		ctxt.RequestCache[apiservice.PatientID] = patientID

		doctorID, err := l.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.DoctorID] = doctorID

		// ensure that the doctor has access to the patient information
		if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorID, patientID, l.dataAPI); err != nil {
			return false, err
		}

	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientID := ctxt.RequestCache[apiservice.PatientID].(int64)

	cases, err := l.dataAPI.GetCasesForPatient(patientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &listCasesResponseData{PatientCases: cases})
}
