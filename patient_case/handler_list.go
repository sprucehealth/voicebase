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
	PatientId int64 `schema:"patient_id"`
}

type listCasesResponseData struct {
	PatientCases []*common.PatientCase `json:"cases"`
}

func NewListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&listHandler{
		dataAPI: dataAPI,
	}, []string{apiservice.HTTP_GET})
}

func (l *listHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	switch ctxt.Role {
	case api.PATIENT_ROLE:
		patientId, err := l.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientId] = patientId

	case api.DOCTOR_ROLE:
		var requestData listCasesRequestData
		if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
			return false, apiservice.NewValidationError(err.Error(), r)
		}
		patientId := requestData.PatientId
		ctxt.RequestCache[apiservice.PatientId] = patientId

		doctorId, err := l.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.DoctorId] = doctorId

		// ensure that the doctor has access to the patient information
		if err := apiservice.ValidateDoctorAccessToPatientFile(doctorId, patientId, l.dataAPI); err != nil {
			return false, err
		}

	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientId := ctxt.RequestCache[apiservice.PatientId].(int64)

	cases, err := l.dataAPI.GetCasesForPatient(patientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &listCasesResponseData{PatientCases: cases})
}
