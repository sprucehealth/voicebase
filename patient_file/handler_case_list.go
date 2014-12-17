package patient_file

import (
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type caseListResponse struct {
	Cases []*caseItem `json:"cases"`
}

type caseItem struct {
	ID            int64               `json:"id,string"`
	Title         string              `json:"title"`
	Status        string              `json:"status"`
	PatientVisits []*patientVisitItem `json:"patient_visits"`
	ActiveTPs     []*tpItem           `json:"active_treatment_plans,omitempty"`
	InactiveTPs   []*tpItem           `json:"inactive_treatment_plans,omitempty"`
	DraftTPs      []*tpItem           `json:"draft_treatment_plans,omitempty"`
}

type patientVisitItem struct {
	ID            int64     `json:"id,string"`
	Title         string    `json:"title"`
	SubmittedDate time.Time `json:"submitted_date"`
	Status        string    `json:"status"`
}

type tpItem struct {
	ID            int64            `json:"id,string"`
	DoctorID      int64            `json:"doctor_id,string"`
	Status        string           `json:"status"`
	CreationDate  time.Time        `json:"creation_date"`
	Parent        *tpParent        `json:"parent,omitempty"`
	ContentSource *tpContentSource `json:"content_source,omitempty"`
}

type tpParent struct {
	ID           int64     `json:"parent_id,string"`
	Type         string    `json:"parent_type"`
	CreationDate time.Time `json:"creation_date"`
}

type tpContentSource struct {
	ID       int64  `json:"content_source_id,string"`
	Type     string `json:"content_source_type"`
	Deviated bool   `json:"has_deviated"`
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
		return false, apiservice.NewValidationError(err.Error(), r)
	} else if rd.PatientID == 0 {
		return false, apiservice.NewValidationError("patient_id required", r)
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
	caseList := make([]*caseItem, len(cases))
	for i, pc := range cases {

		// FIXME: Fix hardcoded values for the title and status of the case
		item := &caseItem{
			ID:     pc.ID.Int64(),
			Title:  "Acne Case",
			Status: "ACTIVE",
		}
		caseList[i] = item

		// get the visits for the case
		visits, err := c.dataAPI.GetVisitsForCase(pc.ID.Int64(), common.NonOpenPatientVisitStates())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		item.PatientVisits = make([]*patientVisitItem, len(visits))
		for j, visit := range visits {
			var title string
			if visit.IsFollowup {
				title = "Follow-up Visit"
			} else {
				title = "Initial Visit"
			}

			item.PatientVisits[j] = &patientVisitItem{
				ID:            visit.PatientVisitID.Int64(),
				Title:         title,
				Status:        visit.Status,
				SubmittedDate: visit.SubmittedDate,
			}
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

func populateTPList(tps []*common.TreatmentPlan) []*tpItem {
	tpList := make([]*tpItem, len(tps))
	for i, tp := range tps {
		item := &tpItem{
			ID:           tp.ID.Int64(),
			Status:       tp.Status.String(),
			DoctorID:     tp.DoctorID.Int64(),
			CreationDate: tp.CreationDate,
		}
		if tp.Parent != nil {
			item.Parent = &tpParent{
				ID:           tp.Parent.ParentID.Int64(),
				Type:         tp.Parent.ParentType,
				CreationDate: tp.Parent.CreationDate,
			}
		}
		if tp.ContentSource != nil {
			item.ContentSource = &tpContentSource{
				ID:       tp.ContentSource.ID.Int64(),
				Type:     tp.ContentSource.Type,
				Deviated: tp.ContentSource.HasDeviated,
			}
		}
		tpList[i] = item
	}

	return tpList
}
