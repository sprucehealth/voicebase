package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
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

func NewResourceGuideHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(&resourceGuideHandler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher,
				})),
			api.RoleDoctor,
		),
		httputil.Get, httputil.Put, httputil.Delete)
}

func (h *resourceGuideHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	req := &ResourceGuideRequest{}
	if err := apiservice.DecodeRequestData(req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if req.TreatmentPlanID == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified")
	}
	requestCache[apiservice.CKRequestData] = req

	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	patientID, err := h.dataAPI.GetPatientIDFromTreatmentPlanID(req.TreatmentPlanID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientID] = patientID

	tp, err := h.dataAPI.GetAbridgedTreatmentPlan(req.TreatmentPlanID, doctorID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKTreatmentPlan] = tp

	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, patientID, tp.PatientCaseID.Int64(), h.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (h *resourceGuideHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	tp := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

	if !tp.InDraftMode() {
		apiservice.WriteValidationError(ctx, "treatment plan must be in draft mode", w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.listResourceGuides(ctx, w, r)
	case "PUT":
		h.addResourceGuides(ctx, w, r)
	case "DELETE":
		h.removeResourceGuide(ctx, w, r)
	}
}

func (h *resourceGuideHandler) listResourceGuides(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*ResourceGuideRequest)

	guides, err := h.dataAPI.ListTreatmentPlanResourceGuides(req.TreatmentPlanID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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

func (h *resourceGuideHandler) addResourceGuides(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*ResourceGuideRequest)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	ids := make([]int64, len(req.GuideIDs))
	for i, id := range req.GuideIDs {
		ids[i] = id.Int64()
	}
	if err := h.dataAPI.AddResourceGuidesToTreatmentPlan(req.TreatmentPlanID, ids); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		DoctorID:        doctorID,
		TreatmentPlanID: req.TreatmentPlanID,
		SectionUpdated:  ResourceGuidesSection,
	})

	apiservice.WriteJSONSuccess(w)
}

func (h *resourceGuideHandler) removeResourceGuide(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	req := requestCache[apiservice.CKRequestData].(*ResourceGuideRequest)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)

	if err := h.dataAPI.RemoveResourceGuidesFromTreatmentPlan(req.TreatmentPlanID, []int64{req.GuideID}); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	h.dispatcher.Publish(&TreatmentPlanUpdatedEvent{
		DoctorID:        doctorID,
		TreatmentPlanID: req.TreatmentPlanID,
		SectionUpdated:  ResourceGuidesSection,
	})

	apiservice.WriteJSONSuccess(w)
}
