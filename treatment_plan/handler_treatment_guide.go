package treatment_plan

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
)

var footerText = `This prescription guide covers only common use and is not meant to be a complete listing of drug information. If you are experiencing concerning symptoms, seek medical attention immediately.

For more information, please see the package insert that came with your medication or ask your pharmacist or physician directly.`

type TreatmentGuideRequestData struct {
	TreatmentID int64 `schema:"treatment_id,required"`
}

type treatmentGuideHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentGuideHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&treatmentGuideHandler{
						dataAPI: dataAPI,
					})),
			api.RolePatient, api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (h *treatmentGuideHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestData := new(TreatmentGuideRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = requestData

	treatment, err := h.dataAPI.GetTreatmentFromID(requestData.TreatmentID)
	if err != nil {
		return false, err
	} else if treatment == nil {
		return false, apiservice.NewResourceNotFoundError("treatment not found", r)
	}
	requestCache[apiservice.CKTreatment] = treatment

	treatmentPlan, err := h.dataAPI.GetTreatmentPlanForPatient(treatment.PatientID.Int64(), treatment.TreatmentPlanID.Int64())
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

	account := apiservice.MustCtxAccount(ctx)
	switch account.Role {
	case api.RolePatient:
		patientID, err := h.dataAPI.GetPatientIDFromAccountID(account.ID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKPatientID] = patientID

		if treatment.PatientID.Int64() != patientID {
			return false, apiservice.NewAccessForbiddenError()
		}

	case api.RoleDoctor:
		doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(account.ID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKDoctorID] = doctorID

		if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID, treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), h.dataAPI); err != nil {
			return false, err
		}

		// ensure that doctor is owner of the treatment plan
		if doctorID != treatmentPlan.DoctorID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}
	}

	return true, nil
}

func (h *treatmentGuideHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	treatment := requestCache[apiservice.CKTreatment].(*common.Treatment)
	treatmentPlan := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

	treatmentGuideResponse(ctx, h.dataAPI, treatment.GenericDrugName, treatment.DrugRoute, treatment.DrugForm, treatment.DosageStrength, treatment.DrugDBIDs[erx.NDC], treatment, treatmentPlan, w, r)
}

func treatmentGuideResponse(ctx context.Context, dataAPI api.DataAPI, genericName, route, form, dosage, ndc string, treatment *common.Treatment, treatmentPlan *common.TreatmentPlan, w http.ResponseWriter, r *http.Request) {
	details, err := dataAPI.QueryDrugDetails(&api.DrugDetailsQuery{
		NDC:         ndc,
		GenericName: genericName,
		Route:       route,
		Form:        form,
	})
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, "No details available", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	tgViews, err := treatmentGuideViews(details, dosage, treatment, treatmentPlan)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, map[string][]views.View{"views": tgViews})
}

func treatmentGuideViews(details *common.DrugDetails, dosage string, treatment *common.Treatment, treatmentPlan *common.TreatmentPlan) ([]views.View, error) {
	var tgViews []views.View

	name := details.Name
	if treatment != nil {
		name = fmt.Sprintf("%s %s %s", details.Name, dosage, details.Form)
	}
	tgViews = append(tgViews,
		&tpIconTitleSubtitleView{
			Title:    name,
			Subtitle: details.OtherNames,
		},
		&views.SmallDivider{},
		&views.Text{
			Text: details.Description,
		},
	)

	if treatment != nil || len(details.Tips) != 0 {
		tgViews = append(tgViews,
			&views.LargeDivider{},
			&views.Text{
				Text:  "Instructions",
				Style: views.SectionHeaderStyle,
			},
		)

		if treatment != nil {
			tgViews = append(tgViews,
				&views.SmallDivider{},
				&views.Text{
					Text:  strings.ToUpper(fmt.Sprintf("%s's Instructions", treatment.Doctor.ShortDisplayName)),
					Style: views.SubheaderStyle,
				},
				&views.Text{
					Text: treatment.PatientInstructions,
				},
			)
		}

		if len(details.Tips) != 0 {
			tgViews = append(tgViews,
				&views.SmallDivider{},
				&views.Text{
					Text:  "TIPS",
					Style: views.SubheaderStyle,
				},
			)
			for _, t := range details.Tips {
				tgViews = append(tgViews,
					&views.Text{
						Text: t,
					},
				)
			}
		}
	}

	if len(details.Warnings) != 0 {
		tgViews = append(tgViews,
			&views.LargeDivider{},
			&views.Text{
				Text:  "Warnings",
				Style: views.SectionHeaderStyle,
			},
			&views.SmallDivider{},
		)
		for _, s := range details.Warnings {
			tgViews = append(tgViews, &views.Text{
				Text: s,
			})
		}
	}

	if len(details.CommonSideEffects) != 0 {
		tgViews = append(tgViews,
			&views.LargeDivider{},
			&views.Text{
				Text:  "Common Side Effects",
				Style: views.SectionHeaderStyle,
			},
			&views.SmallDivider{},
		)
		for _, s := range details.CommonSideEffects {
			tgViews = append(tgViews, &views.Text{
				Text: s,
			})
		}
	}

	if treatment != nil && treatmentPlan != nil {
		tgViews = append(tgViews,
			&tpButtonFooterView{
				FooterText: footerText,
				ButtonText: "Message Care Team",
				IconURL:    app_url.IconMessage,
				TapURL:     app_url.SendCaseMessageAction(treatmentPlan.PatientCaseID.Int64()),
			},
		)
	} else {
		tgViews = append(tgViews,
			&tpButtonFooterView{
				FooterText: footerText,
			},
		)
	}

	return tgViews, views.Validate(tgViews, treatmentViewNamespace)
}
