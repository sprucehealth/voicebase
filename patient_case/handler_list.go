package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
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

func NewListHandler(dataAPI api.DataAPI) *listHandler {
	return &listHandler{
		dataAPI: dataAPI,
	}
}

func (l *listHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	ctx := apiservice.GetContext(r)

	var patientId int64
	switch ctx.Role {
	case api.DOCTOR_ROLE:
		var requestData listCasesRequestData
		if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
		patientId = requestData.PatientId

		doctorId, err := l.dataAPI.GetDoctorIdFromAccountId(ctx.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// ensure that the doctor has access to the patient information
		if err := apiservice.ValidateDoctorAccessToPatientFile(doctorId, patientId, l.dataAPI); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	case api.PATIENT_ROLE:
		var err error
		patientId, err = l.dataAPI.GetPatientIdFromAccountId(ctx.AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	cases, err := l.dataAPI.GetCasesForPatient(patientId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &listCasesResponseData{PatientCases: cases})
}
