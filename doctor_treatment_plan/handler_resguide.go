package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
)

type resourceGuideHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type ResourceGuideRequest struct {
	TreatmentPlanID int64               `json:"treatment_plan_id,string" schema:"treatment_plan_id"`
	GuideID         int64               `json:"resource_guide_id,string,omitempty" schema:"resource_guide_id"`
	GuideIDs        []encoding.ObjectID `json:"resource_guide_ids,omitempty"`
}

type ResourceGuideResponse struct {
	Guides []*responses.ResourceGuide `json:"resource_guides"`
}

func NewResourceGuideHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(&resourceGuideHandler{
				dataAPI:    dataAPI,
				dispatcher: dispatcher,
			}),
			[]string{api.RoleDoctor},
		),
		[]string{"GET", "PUT", "DELETE"})
}

func (h *resourceGuideHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	req := &ResourceGuideRequest{}
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if req.TreatmentPlanID == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified")
	}
	ctxt.RequestCache[apiservice.RequestData] = req

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	patientID, err := h.dataAPI.GetPatientIDFromTreatmentPlanID(req.TreatmentPlanID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientID] = patientID

	tp, err := h.dataAPI.GetAbridgedTreatmentPlan(req.TreatmentPlanID, doctorID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = tp

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID, patientID, tp.PatientCaseID.Int64(), h.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (h *resourceGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	tp := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	if !tp.InDraftMode() {
		apiservice.WriteValidationError("treatment plan must be in draft mode", w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.listResourceGuides(w, r)
	case "PUT":
		h.addResourceGuides(w, r)
	case "DELETE":
		h.removeResourceGuide(w, r)
	}
}

func (h *resourceGuideHandler) listResourceGuides(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	req := ctxt.RequestCache[apiservice.RequestData].(*ResourceGuideRequest)

	guides, err := h.dataAPI.ListTreatmentPlanResourceGuides(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res := &ResourceGuideResponse{
		Guides: make([]*responses.ResourceGuide, len(guides)),
	}
	for i, g := range guides {
		res.Guides[i] = &responses.ResourceGuide{
			ID:        g.ID,
			SectionID: g.SectionID,
			Title:     g.Title,
			PhotoURL:  g.PhotoURL,
		}
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func (h *resourceGuideHandler) addResourceGuides(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*ResourceGuideRequest)
	ids := make([]int64, len(req.GuideIDs))
	doctorID := ctx.RequestCache[apiservice.DoctorID].(int64)
	for i, id := range req.GuideIDs {
		ids[i] = id.Int64()
	}
	if err := h.dataAPI.AddResourceGuidesToTreatmentPlan(req.TreatmentPlanID, ids); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		DoctorID:        doctorID,
		TreatmentPlanID: req.TreatmentPlanID,
		SectionUpdated:  ResourceGuidesSection,
	})

	apiservice.WriteJSONSuccess(w)
}

func (h *resourceGuideHandler) removeResourceGuide(w http.ResponseWriter, r *http.Request) {
	ctx := apiservice.GetContext(r)
	req := ctx.RequestCache[apiservice.RequestData].(*ResourceGuideRequest)
	doctorID := ctx.RequestCache[apiservice.DoctorID].(int64)

	if err := h.dataAPI.RemoveResourceGuidesFromTreatmentPlan(req.TreatmentPlanID, []int64{req.GuideID}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		DoctorID:        doctorID,
		TreatmentPlanID: req.TreatmentPlanID,
		SectionUpdated:  ResourceGuidesSection,
	})

	apiservice.WriteJSONSuccess(w)
}
