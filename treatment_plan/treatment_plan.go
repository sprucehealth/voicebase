package treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
)

type treatmentPlanHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentPlanHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&treatmentPlanHandler{
					dataAPI: dataAPI,
				}), []string{api.RolePatient, api.RoleDoctor}),
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
}

func (p *treatmentPlanHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &TreatmentPlanRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	switch ctxt.Role {
	case api.RolePatient:
		if requestData.TreatmentPlanID == 0 && requestData.PatientCaseID == 0 {
			return false, apiservice.NewValidationError("either treatment_plan_id or patient_case_id must be specified")
		}

		patient, err := p.dataAPI.GetPatientFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.Patient] = patient

		var treatmentPlan *common.TreatmentPlan
		if requestData.TreatmentPlanID != 0 {
			treatmentPlan, err = p.dataAPI.GetTreatmentPlanForPatient(patient.ID.Int64(), requestData.TreatmentPlanID)
		} else {
			treatmentPlan, err = p.dataAPI.GetActiveTreatmentPlanForCase(requestData.PatientCaseID)
		}
		if api.IsErrNotFound(err) {
			return false, apiservice.NewResourceNotFoundError("treatment plan not found", r)
		} else if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if treatmentPlan.PatientID != patient.ID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

		if !treatmentPlan.IsReadyForPatient() {
			return false, apiservice.NewResourceNotFoundError("Inactive/active treatment_plan not found", r)
		}

		doctor, err := p.dataAPI.GetDoctorFromID(treatmentPlan.DoctorID.Int64())
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.Doctor] = doctor

	case api.RoleDoctor:
		if requestData.TreatmentPlanID == 0 {
			return false, apiservice.NewValidationError("treatment_plan_id must be specified")
		}

		doctor, err := p.dataAPI.GetDoctorFromAccountID(ctxt.AccountID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.Doctor] = doctor

		patient, err := p.dataAPI.GetPatientFromTreatmentPlanID(requestData.TreatmentPlanID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.Patient] = patient

		treatmentPlan, err := p.dataAPI.GetTreatmentPlanForPatient(patient.ID.Int64(), requestData.TreatmentPlanID)
		if api.IsErrNotFound(err) {
			return false, apiservice.NewResourceNotFoundError("treatment plan not found", r)
		} else if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		if err = apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctor.ID.Int64(), patient.ID.Int64(),
			treatmentPlan.PatientCaseID.Int64(), p.dataAPI); err != nil {
			return false, err
		}
	default:
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

func (p *treatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

	err := populateTreatmentPlan(p.dataAPI, treatmentPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	res, err := treatmentPlanResponse(p.dataAPI, treatmentPlan, doctor, patient)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}

func treatmentPlanResponse(dataAPI api.DataAPI, tp *common.TreatmentPlan, doctor *common.Doctor, patient *common.Patient) (*TreatmentPlanViewsResponse, error) {
	var headerViews, treatmentViews, instructionViews []views.View

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

	// TREATMENT VIEWS
	if len(tp.TreatmentList.Treatments) > 0 {
		treatmentViews = append(treatmentViews, GenerateViewsForTreatments(tp.TreatmentList, tp.ID.Int64(), dataAPI, false)...)
		cardViews := []views.View{
			&tpCardTitleView{
				Title: "How to get your treatments",
			},
		}
		hasRX := false
		hasOTC := false
		for _, t := range tp.TreatmentList.Treatments {
			if t.OTC {
				hasOTC = true
			} else {
				hasRX = true
			}
		}
		if hasRX {
			cardViews = append(cardViews,
				&tpTextView{
					Text:  "Prescription",
					Style: views.SubheaderStyle,
				},
				&tpTextView{
					Text: "Your prescriptions have been sent to your pharmacy. We suggest calling ahead to ask about price. If it seems expensive, message your care coordinator for help.",
				},
			)
		}
		if hasOTC {
			cardViews = append(cardViews,
				&tpTextView{
					Text:  "Over-the-counter",
					Style: views.SubheaderStyle,
				},
				&tpTextView{
					Text: "Check with your pharmacist before looking for your over-the-counter treatment in the aisles. OTC treatments may be less expensive when purchased through the pharmacy.",
				},
			)
		}
		cardViews = append(cardViews,
			&tpTextView{
				Text:  "Your pharmacy",
				Style: views.SubheaderStyle,
			},
			&tpPharmacyView{
				Text:     "Your prescriptions should be ready soon. Call your pharmacy to confirm a pickup time.",
				Pharmacy: patient.Pharmacy,
			},
		)
		treatmentViews = append(treatmentViews,
			&tpCardView{
				Views: cardViews,
			},
			&tpButtonFooterView{
				FooterText:       fmt.Sprintf("If you have any questions about your treatment plan, message your care team."),
				ButtonText:       "Send a Message",
				IconURL:          app_url.IconMessage,
				TapURL:           app_url.SendCaseMessageAction(tp.PatientCaseID.Int64()),
				CenterFooterText: true,
			},
		)
	}

	// INSTRUCTION VIEWS
	if tp.RegimenPlan != nil && len(tp.RegimenPlan.Sections) > 0 {
		for _, regimenSection := range tp.RegimenPlan.Sections {
			cView := &tpCardView{
				Views: []views.View{},
			}
			instructionViews = append(instructionViews, cView)

			cView.Views = append(cView.Views, &tpCardTitleView{
				Title: regimenSection.Name,
			})

			for _, regimenStep := range regimenSection.Steps {
				cView.Views = append(cView.Views, &tpListElementView{
					ElementStyle: bulletedStyle,
					Text:         regimenStep.Text,
				})
			}
		}
	}

	if len(tp.ResourceGuides) != 0 {
		rgViews := []views.View{
			&tpCardTitleView{
				Title: "Resources",
			},
		}
		for i, g := range tp.ResourceGuides {
			if i != 0 {
				rgViews = append(rgViews, &views.SmallDivider{})
			}
			rgViews = append(rgViews, &tpLargeIconTextButtonView{
				Text:       g.Title,
				IconURL:    g.PhotoURL,
				IconWidth:  66,
				IconHeight: 66,
				TapURL:     app_url.ViewResourceGuideAction(g.ID),
			})
		}
		instructionViews = append(instructionViews, &tpCardView{
			Views: rgViews,
		})
	}

	instructionViews = append(instructionViews, &tpButtonFooterView{
		FooterText:       "If you have any questions about your treatment plan, message your care team.",
		ButtonText:       "Send a Message",
		IconURL:          app_url.IconMessage,
		TapURL:           app_url.SendCaseMessageAction(tp.PatientCaseID.Int64()),
		CenterFooterText: true,
	})

	for _, vContainer := range [][]views.View{headerViews, treatmentViews, instructionViews} {
		if err := views.Validate(vContainer, treatmentViewNamespace); err != nil {
			return nil, err
		}
	}

	return &TreatmentPlanViewsResponse{
		HeaderViews:      headerViews,
		TreatmentViews:   treatmentViews,
		InstructionViews: instructionViews,
	}, nil
}
