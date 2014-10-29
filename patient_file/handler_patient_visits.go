package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type patientVisitsHandler struct {
	DataApi api.DataAPI
}

type request struct {
	PatientID int64 `schema:"patient_id,required"`
	CaseID    int64 `schema:"case_id"`
}

type response struct {
	PatientVisits []*common.PatientVisit `json:"patient_visits"`
}

func NewPatientVisitsHandler(dataApi api.DataAPI) http.Handler {
	return &patientVisitsHandler{
		DataApi: dataApi,
	}
}

func (p *patientVisitsHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	requestData := &request{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := p.DataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patient, err := p.DataApi.GetPatientFromId(requestData.PatientID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctor.DoctorId.Int64(), patient.PatientId.Int64(), p.DataApi); err != nil {
		return false, err
	}

	return true, nil
}

func (p *patientVisitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*request)
	var patientCase *common.PatientCase
	if requestData.CaseID == 0 {
		cases, err := p.DataApi.GetCasesForPatient(patient.PatientId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if len(cases) == 0 {
			apiservice.WriteValidationError("no cases exist for patient", w, r)
			return
		}

		// pick the first case for the patient if the caseID is not specified
		patientCase = cases[0]
	} else {
		var err error
		patientCase, err = p.DataApi.GetPatientCaseFromId(requestData.CaseID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	states := common.TreatedPatientVisitStates()
	states = append(states, common.SubmittedPatientVisitStates()...)
	patientVisits, err := p.DataApi.GetVisitsForCase(patientCase.Id.Int64(), states)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, response{PatientVisits: patientVisits})
}
