package patient_file

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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

func NewPatientVisitsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&patientVisitsHandler{
				DataAPI: dataAPI,
			})),
		httputil.Get)
}

func (p *patientVisitsHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	requestData := &request{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	doctor, err := p.DataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	patient, err := p.DataAPI.GetPatientFromID(requestData.PatientID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatient] = patient

	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, account.Role, doctor.ID.Int64(), patient.ID.Int64(), p.DataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (p *patientVisitsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patient := requestCache[apiservice.CKPatient].(*common.Patient)
	requestData := requestCache[apiservice.CKRequestData].(*request)
	var patientCase *common.PatientCase
	if requestData.CaseID == 0 {
		cases, err := p.DataAPI.GetCasesForPatient(patient.ID.Int64(), []string{common.PCStatusActive.String(), common.PCStatusInactive.String()})
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if len(cases) == 0 {
			apiservice.WriteValidationError(ctx, "no cases exist for patient", w, r)
			return
		}

		// pick the first case for the patient if the caseID is not specified
		patientCase = cases[0]
	} else {
		var err error
		patientCase, err = p.DataAPI.GetPatientCaseFromID(requestData.CaseID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	states := common.TreatedPatientVisitStates()
	states = append(states, common.SubmittedPatientVisitStates()...)
	patientVisits, err := p.DataAPI.GetVisitsForCase(patientCase.ID.Int64(), states)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
