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

// HACK: patientVisitItem embeds the patientVisit struct and then adds the health_condition_id
// which is something that the doctor clients pre-buzz depend on
type patientVisitItem struct {
	*common.PatientVisit
	DeprecatedHealthConditionID int64 `json:"health_condition_id,string"`
}

type response struct {
	PatientVisits []*patientVisitItem `json:"patient_visits"`
}

func NewPatientVisitsHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&patientVisitsHandler{
			DataAPI: dataAPI,
		}), httputil.Get)
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

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctor.ID.Int64(), patient.ID.Int64(), p.DataAPI); err != nil {
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
		cases, err := p.DataAPI.GetCasesForPatient(patient.ID.Int64(), []string{common.PCStatusActive.String(), common.PCStatusInactive.String()})
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

	visits := make([]*patientVisitItem, len(patientVisits))
	for i, visit := range patientVisits {

		visits[i] = &patientVisitItem{
			PatientVisit:                visit,
			DeprecatedHealthConditionID: 1,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response{PatientVisits: visits})
}
