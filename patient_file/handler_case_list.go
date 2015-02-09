package patient_file

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type caseListResponse struct {
	Cases []*responses.Case `json:"cases"`
}

type caseListHandler struct {
	dataAPI api.DataAPI
}

type caseListRequest struct {
	PatientID int64 `schema:"patient_id,required"`
}

func NewPatientCaseListHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&caseListHandler{
					dataAPI: dataAPI,
				}), []string{api.DOCTOR_ROLE, api.MA_ROLE}), []string{"GET"})
}

func (c *caseListHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	rd := &caseListRequest{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if rd.PatientID == 0 {
		return false, apiservice.NewValidationError("patient_id required")
	}
	ctxt.RequestCache[apiservice.RequestData] = rd

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	// ensure doctor/ma has access to read patient file
	if err := apiservice.ValidateDoctorAccessToPatientFile(r.Method, ctxt.Role, doctorID, rd.PatientID, c.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (c *caseListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	rd := ctxt.RequestCache[apiservice.RequestData].(*caseListRequest)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)

	// get a list of cases for the patient
	cases, err := c.dataAPI.GetCasesForPatient(rd.PatientID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// populate list of cases
	caseList := make([]*responses.Case, len(cases))
	for i, pc := range cases {

		// FIXME: Fix hardcoded values for the status of the case
		item := &responses.Case{
			ID:         pc.ID.Int64(),
			Title:      fmt.Sprintf("%s Case", pc.Name),
			PathwayTag: pc.PathwayTag,
			Status:     "ACTIVE",
		}
		caseList[i] = item

		// get the visits for the case
		visits, err := c.dataAPI.GetVisitsForCase(pc.ID.Int64(), common.NonOpenPatientVisitStates())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		item.PatientVisits = make([]*responses.PatientVisit, len(visits))
		for j, visit := range visits {
			var title string
			if visit.IsFollowup {
				title = "Follow-up Visit"
			} else {
				title = "Initial Visit"
			}

			item.PatientVisits[j] = responses.NewPatientVisit(visit, title)
		}

		activeTPs, err := c.dataAPI.GetAbridgedTreatmentPlanList(doctorID, rd.PatientID, common.ActiveTreatmentPlanStates())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		item.ActiveTPs = populateTPList(activeTPs)

		inactiveTPs, err := c.dataAPI.GetAbridgedTreatmentPlanList(doctorID, rd.PatientID, common.InactiveTreatmentPlanStates())
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}
		item.InactiveTPs = populateTPList(inactiveTPs)

		draftTreatmentPlans, err := c.dataAPI.GetAbridgedTreatmentPlanListInDraftForDoctor(doctorID, rd.PatientID)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
			return
		}
		item.DraftTPs = populateTPList(draftTreatmentPlans)
	}

	apiservice.WriteJSON(w, caseListResponse{
		Cases: caseList,
	})
}

func populateTPList(tps []*common.TreatmentPlan) []*responses.TreatmentPlan {
	tpList := make([]*responses.TreatmentPlan, len(tps))
	for i, tp := range tps {
		item := responses.NewTreatmentPlan(tp)
		if tp.Parent != nil {
			item.Parent = responses.NewTreatmentPlanParent(tp.Parent)
		}
		if tp.ContentSource != nil {
			item.ContentSource = &responses.TreatmentPlanContentSource{
				ID:       tp.ContentSource.ID.Int64(),
				Type:     tp.ContentSource.Type,
				Deviated: tp.ContentSource.HasDeviated,
			}
		}
		tpList[i] = item
	}

	return tpList
}
