package doctor_treatment_plan

import (
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type treatmentTemplatesHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentTemplatesHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&treatmentTemplatesHandler{
				dataAPI: dataAPI,
			})),
		httputil.Get, httputil.Post, httputil.Delete)
}

type DoctorTreatmentTemplatesRequest struct {
	TreatmentPlanID    encoding.ObjectID                 `json:"treatment_plan_id"`
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
}

type DoctorTreatmentTemplatesResponse struct {
	TreatmentTemplates []*common.DoctorTreatmentTemplate `json:"treatment_templates"`
	Treatments         []*common.Treatment               `json:"treatments,omitempty"`
}

func (t *treatmentTemplatesHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)
	if account.Role != api.RoleDoctor {
		return false, apiservice.NewAccessForbiddenError()
	}

	if r.Method == httputil.Get {
		return true, nil
	}

	requestData := &DoctorTreatmentTemplatesRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, err
	} else if requestData.TreatmentPlanID.Int64() == 0 {
		return false, apiservice.NewValidationError("treatment_plan_id must be specified")
	}
	requestCache[apiservice.CKRequestData] = requestData

	doctorID, err := t.dataAPI.GetDoctorIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	treatmentPlan, err := t.dataAPI.GetAbridgedTreatmentPlan(requestData.TreatmentPlanID.Int64(), doctorID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), t.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (t *treatmentTemplatesHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Get:
		t.getTreatmentTemplates(ctx, w, r)
	case httputil.Post:
		t.addTreatmentTemplates(ctx, w, r)
	case httputil.Delete:
		t.deleteTreatmentTemplates(ctx, w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (t *treatmentTemplatesHandler) getTreatmentTemplates(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := t.dataAPI.GetDoctorIDFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to get doctor from account id: "+err.Error()), w, r)
		return
	}

	doctorTreatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorID)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to get favorite treatments for doctor: "+err.Error()), w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{TreatmentTemplates: doctorTreatmentTemplates})
}

func (t *treatmentTemplatesHandler) deleteTreatmentTemplates(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	requestData := requestCache[apiservice.CKRequestData].(*DoctorTreatmentTemplatesRequest)

	for _, favoriteTreatment := range requestData.TreatmentTemplates {
		if favoriteTreatment.ID.Int64() == 0 {
			apiservice.WriteValidationError(ctx, "Unable to delete a treatment that does not have an id associated with it", w, r)
			return
		}
	}

	err := t.dataAPI.DeleteTreatmentTemplates(requestData.TreatmentTemplates, doctorID)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to delete favorited treatment: "+err.Error()), w, r)
		return
	}

	treatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorID)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to get favorite treatments for doctor: "+err.Error()), w, r)
		return
	}

	treatmentsInTreatmentPlan, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to get treatments based on treatment plan id: "+err.Error()), w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}

func (t *treatmentTemplatesHandler) addTreatmentTemplates(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	requestData := requestCache[apiservice.CKRequestData].(*DoctorTreatmentTemplatesRequest)

	for _, treatmentTemplate := range requestData.TreatmentTemplates {
		err := apiservice.ValidateTreatment(treatmentTemplate.Treatment)
		if err != nil {
			apiservice.WriteValidationError(ctx, err.Error(), w, r)
			return
		}

		// break up the name into its components so that it can be saved into the database as its components
		treatmentTemplate.Treatment.DrugName, treatmentTemplate.Treatment.DrugForm, treatmentTemplate.Treatment.DrugRoute = apiservice.BreakDrugInternalNameIntoComponents(treatmentTemplate.Treatment.DrugInternalName)
	}

	err := t.dataAPI.AddTreatmentTemplates(requestData.TreatmentTemplates, doctorID, requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	treatmentTemplates, err := t.dataAPI.GetTreatmentTemplates(doctorID)
	if err != nil {
		apiservice.WriteError(ctx, errors.New("Unable to get favorited treatments for doctor: "+err.Error()), w, r)
		return
	}

	treatmentsInTreatmentPlan, err := t.dataAPI.GetTreatmentsBasedOnTreatmentPlanID(requestData.TreatmentPlanID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if err := indicateExistenceOfRXGuidesForTreatments(t.dataAPI, &common.TreatmentList{
		Treatments: treatmentsInTreatmentPlan,
	}); err != nil {
		golog.Errorf(err.Error())
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorTreatmentTemplatesResponse{
		TreatmentTemplates: treatmentTemplates,
		Treatments:         treatmentsInTreatmentPlan,
	})
}
