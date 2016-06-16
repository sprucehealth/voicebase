package patient_file

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type alertsHandler struct {
	dataAPI api.DataAPI
}

func NewAlertsHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&alertsHandler{
						dataAPI: dataAPI,
					})),
			api.RoleDoctor),
		httputil.Get)

}

type alertsRequestData struct {
	PatientID common.PatientID `schema:"patient_id,required"`
	CaseID    int64            `schema:"case_id"`
	VisitID   int64            `schema:"patient_visit_id"`
}

type alertsResponse struct {
	Alerts []*responses.Alert `json:"alerts"`
}

func (a *alertsHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	requestData := &alertsRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	if !requestData.PatientID.IsValid && requestData.CaseID == 0 && requestData.VisitID == 0 {
		return false, apiservice.NewValidationError("patient_id or case_id or patient_visit_id must be specified")
	}

	doctorID, err := a.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}

	switch {
	case requestData.CaseID > 0:
		pc, err := a.dataAPI.GetPatientCaseFromID(requestData.CaseID)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			account.Role,
			doctorID,
			pc.PatientID,
			pc.ID.Int64(),
			a.dataAPI); err != nil {
			return false, err
		}
	case requestData.VisitID > 0:
		visit, err := a.dataAPI.GetPatientVisitFromID(requestData.VisitID)
		if err != nil {
			return false, err
		}

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			account.Role,
			doctorID,
			visit.PatientID,
			visit.PatientCaseID.Int64(),
			a.dataAPI); err != nil {
			return false, err
		}
	case requestData.PatientID.IsValid:
		if err := apiservice.ValidateDoctorAccessToPatientFile(
			r.Method,
			account.Role,
			doctorID,
			requestData.PatientID,
			a.dataAPI); err != nil {
			return false, err
		}
	}

	requestCache[apiservice.CKRequestData] = requestData

	return true, nil
}

func (a *alertsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)
	rd := requestCache[apiservice.CKRequestData].(*alertsRequestData)

	visitID := rd.VisitID
	var patient *common.Patient
	var err error

	patient, err = a.dataAPI.Patient(rd.PatientID, true)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// If the patient isn't a registered patient then we have no visits or alerts for them currently
	if patient.Status != api.PatientRegistered {
		apiservice.WriteResourceNotFoundError(ctx, "no alerts available for non registered patients", w, r)
		return
	}

	// fall back to caseID or patientID if the visitID is not specified
	switch {
	case visitID > 0:
	case rd.CaseID > 0:
		// get the alerts for the latest visitID pertaining to the case

		visitID, err = a.getVisitIDFromCaseID(rd.CaseID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

	case rd.PatientID.IsValid:
		// get the alerts for the latest visitID of the latest submitted case for the patient

		cases, err := a.dataAPI.GetCasesForPatient(rd.PatientID, common.SubmittedPatientCaseStates())
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		if len(cases) > 0 {
			sort.Sort(sort.Reverse(common.ByPatientCaseCreationDate(cases)))
			caseID := cases[0].ID.Int64()

			visitID, err = a.getVisitIDFromCaseID(caseID)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
		}
	}

	alerts, err := a.dataAPI.AlertsForVisit(visitID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	visit, err := a.dataAPI.GetPatientVisitFromID(visitID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	patientCase, err := a.dataAPI.GetPatientCaseFromID(visit.PatientCaseID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	doctor, err := a.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if patient == nil {
		patient, err = a.dataAPI.GetPatientFromPatientVisitID(visitID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	dAlerts, err := DynamicAlerts(patientCase, doctor, patient, a.dataAPI)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	alerts = append(alerts, dAlerts...)

	response := &alertsResponse{
		Alerts: make([]*responses.Alert, len(alerts)),
	}

	for i, alert := range alerts {
		response.Alerts[i] = responses.TransformAlert(alert)
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

func (a *alertsHandler) getVisitIDFromCaseID(caseID int64) (int64, error) {
	visits, err := a.dataAPI.GetVisitsForCase(caseID, common.NonOpenPatientVisitStates())
	if err != nil {
		return 0, err
	}

	var visitID int64
	if len(visits) > 0 {
		sort.Sort(sort.Reverse(common.ByPatientVisitCreationDate(visits)))
		visitID = visits[0].ID.Int64()
	}

	return visitID, nil
}
