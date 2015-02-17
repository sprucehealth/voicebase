package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type patientVisitsHandler struct {
	DataAPI api.DataAPI
}

type request struct {
	PatientID int64 `schema:"patient_id,required"`
	CaseID    int64 `schema:"case_id"`
}

type response struct {
	PatientVisits []*common.PatientVisit `json:"patient_visits"`
}

func NewPatientVisitsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&patientVisitsHandler{
			DataAPI: dataAPI,
		}), []string{"GET"})
}

func (p *patientVisitsHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &request{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := p.DataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patient, err := p.DataAPI.GetPatientFromID(requestData.PatientID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Patient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctor.DoctorID.Int64(), patient.PatientID.Int64(), p.DataAPI); err != nil {
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
		cases, err := p.DataAPI.GetCasesForPatient(patient.PatientID.Int64())
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
		patientCase, err = p.DataAPI.GetPatientCaseFromID(requestData.CaseID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	states := common.TreatedPatientVisitStates()
	states = append(states, common.SubmittedPatientVisitStates()...)
	patientVisits, err := p.DataAPI.GetVisitsForCase(patientCase.ID.Int64(), states)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, response{PatientVisits: patientVisits})
}
