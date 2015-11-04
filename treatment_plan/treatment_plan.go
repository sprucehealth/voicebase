package treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
	"golang.org/x/net/context"
)

type treatmentPlanHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentPlanHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&treatmentPlanHandler{
						dataAPI: dataAPI,
					})),
			api.RolePatient, api.RoleDoctor),
		httputil.Get)
}

type TreatmentPlanRequest struct {
	TreatmentPlanID int64 `schema:"treatment_plan_id"`
	PatientCaseID   int64 `schema:"case_id"`
}

type TreatmentPlanViewsResponse struct {
	HeaderViews      []views.View `json:"header_views,omitempty"`
	TreatmentViews   []views.View `json:"treatment_views,omitempty"`
	InstructionViews []views.View `json:"instruction_views,omitempty"`
	ContentViews     []views.View `json:"content_views,omitempty"`
}

func (p *treatmentPlanHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestData := &TreatmentPlanRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = requestData

	account := apiservice.MustCtxAccount(ctx)
	switch account.Role {
	case api.RolePatient:
		if requestData.TreatmentPlanID == 0 && requestData.PatientCaseID == 0 {
			return false, apiservice.NewValidationError("either treatment_plan_id or patient_case_id must be specified")
		}

		patient, err := p.dataAPI.GetPatientFromAccountID(account.ID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKPatient] = patient

		var treatmentPlan *common.TreatmentPlan
		if requestData.TreatmentPlanID != 0 {
			treatmentPlan, err = p.dataAPI.GetTreatmentPlanForPatient(patient.ID, requestData.TreatmentPlanID)
		} else {
			treatmentPlan, err = p.dataAPI.GetActiveTreatmentPlanForCase(requestData.PatientCaseID)
		}
		if api.IsErrNotFound(err) {
			return false, apiservice.NewResourceNotFoundError("treatment plan not found", r)
		} else if err != nil {
			return false, err
		}
		requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

		if treatmentPlan.PatientID != patient.ID {
			return false, apiservice.NewAccessForbiddenError()
		}

		if !treatmentPlan.IsReadyForPatient() {
			return false, apiservice.NewResourceNotFoundError("Inactive/active treatment_plan not found", r)
		}

		doctor, err := p.dataAPI.GetDoctorFromID(treatmentPlan.DoctorID.Int64())
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKDoctor] = doctor

	case api.RoleDoctor:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified")
		}

		doctor, err := p.dataAPI.GetDoctorFromAccountID(account.ID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKDoctor] = doctor

		patient, err := p.dataAPI.GetPatientFromTreatmentPlanID(requestData.TreatmentPlanID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKPatient] = patient

		treatmentPlan, err := p.dataAPI.GetTreatmentPlanForPatient(patient.ID, requestData.TreatmentPlanID)
		if api.IsErrNotFound(err) {
			return false, apiservice.NewResourceNotFoundError("treatment plan not found", r)
		} else if err != nil {
			return false, err
		}
		requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

		if err = apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctor.ID.Int64(), patient.ID,
			treatmentPlan.PatientCaseID.Int64(), p.dataAPI); err != nil {
			return false, err
		}
	default:
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (p *treatmentPlanHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctor := requestCache[apiservice.CKDoctor].(*common.Doctor)
	patient := requestCache[apiservice.CKPatient].(*common.Patient)
	treatmentPlan := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

	err := populateTreatmentPlan(p.dataAPI, treatmentPlan)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	res, err := treatmentPlanResponse(ctx, p.dataAPI, treatmentPlan, doctor, patient)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func treatmentPlanResponse(ctx context.Context, dataAPI api.DataAPI, tp *common.TreatmentPlan, doctor *common.Doctor, patient *common.Patient) (*TreatmentPlanViewsResponse, error) {
	var headerViews, treatmentViews, instructionViews, contentViews []views.View

	patientCase, err := dataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	if err != nil {
		return nil, err
	}

	// HEADER VIEWS
	headerViews = append(headerViews,
		&tpHeroHeaderView{
			Title:    fmt.Sprintf("%s's\nTreatment Plan", patient.FirstName),
			Subtitle: fmt.Sprintf("Created by %s\nfor %s", doctor.ShortDisplayName, patientCase.Name),
		})

	p := conc.NewParallel()

	p.Go(func() error {
		treatmentViews, instructionViews = generateViewsForTreatmentsAndInstructions(ctx, tp, patient, dataAPI)
		return nil
	})

	p.Go(func() error {
		contentViews = GenerateViewsForSingleViewTreatmentPlan(ctx, dataAPI, &SingleViewTPConfig{
			TreatmentPlan: tp,
			Pharmacy:      patient.Pharmacy,
		})
		return nil
	})

	if err := p.Wait(); err != nil {
		return nil, err
	}

	// Validate
	for _, vContainer := range [][]views.View{headerViews, treatmentViews, instructionViews, contentViews} {
		if err := views.Validate(vContainer, treatmentViewNamespace); err != nil {
			return nil, err
		}
	}

	return &TreatmentPlanViewsResponse{
		HeaderViews:      headerViews,
		TreatmentViews:   treatmentViews,
		InstructionViews: instructionViews,
		ContentViews:     contentViews,
	}, nil
}
