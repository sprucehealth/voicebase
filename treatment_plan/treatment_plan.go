package treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type treatmentPlanHandler struct {
	dataApi api.DataAPI
}

func NewTreatmentPlanHandler(dataApi api.DataAPI) *treatmentPlanHandler {
	return &treatmentPlanHandler{
		dataApi: dataApi,
	}
}

type TreatmentPlanRequest struct {
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
	PatientCaseId   int64 `schema:"case_id"`
}

type treatmentPlanViewsResponse struct {
	HeaderViews      []tpView `json:"header_views,omitempty"`
	TreatmentViews   []tpView `json:"treatment_views,omitempty"`
	InstructionViews []tpView `json:"instruction_views,omitempty"`
}

func (p *treatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	requestData := &TreatmentPlanRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	var doctor *common.Doctor
	var patient *common.Patient
	var roleId int64
	var treatmentPlan *common.TreatmentPlan
	var err error
	switch apiservice.GetContext(r).Role {
	case api.PATIENT_ROLE:
		if requestData.TreatmentPlanId == 0 && requestData.PatientCaseId == 0 {
			apiservice.WriteValidationError("either treatment_plan_id or patient_case_id must be specified", w, r)
			return
		}

		patient, err = p.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		roleId = patient.PatientId.Int64()

		if requestData.TreatmentPlanId != 0 {
			treatmentPlan, err = p.dataApi.GetTreatmentPlanForPatient(patient.PatientId.Int64(), requestData.TreatmentPlanId)
		} else {
			treatmentPlan, err = p.dataApi.GetActiveTreatmentPlanForCase(requestData.PatientCaseId)
		}

		if err == api.NoRowsError {
			apiservice.WriteResourceNotFoundError("Treatment plan not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		} else if treatmentPlan.Status != api.STATUS_ACTIVE {
			apiservice.WriteResourceNotFoundError("No active treatment plan found for patient", w, r)
			return
		}

		doctor, err = p.dataApi.GetDoctorFromId(treatmentPlan.DoctorId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	case api.DOCTOR_ROLE:

		if requestData.TreatmentPlanId == 0 {
			apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
			return
		}

		doctor, err = p.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		roleId = doctor.DoctorId.Int64()

		patient, err = p.dataApi.GetPatientFromTreatmentPlanId(requestData.TreatmentPlanId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		treatmentPlan, err = p.dataApi.GetTreatmentPlanForPatient(patient.PatientId.Int64(), requestData.TreatmentPlanId)
		if err == api.NoRowsError {
			apiservice.WriteResourceNotFoundError("Treatment plan not found", w, r)
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err = apiservice.ValidateReadAccessToPatientCase(doctor.DoctorId.Int64(), patient.PatientId.Int64(),
			treatmentPlan.PatientCaseId.Int64(), p.dataApi); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	default:
		apiservice.WriteValidationError("Unable to identify role", w, r)
		return
	}

	dispatch.Default.Publish(&TreatmentPlanOpenedEvent{
		RoleType:      apiservice.GetContext(r).Role,
		RoleId:        roleId,
		TreatmentPlan: treatmentPlan,
	})

	err = populateTreatmentPlan(p.dataApi, treatmentPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlanResponse(p.dataApi, w, r, treatmentPlan, doctor, patient)
}

func treatmentPlanResponse(dataApi api.DataAPI, w http.ResponseWriter, r *http.Request, treatmentPlan *common.TreatmentPlan, doctor *common.Doctor, patient *common.Patient) {
	var headerViews, treatmentViews, instructionViews []tpView

	// HEADER VIEWS
	headerViews = append(headerViews,
		&tpHeroHeaderView{
			Title:   fmt.Sprintf("%s's Acne Treatment Plan", patient.FirstName),
			IconURL: app_url.Treatment,
		},
		&tpSmallDividerView{},
		&tpSmallHeaderView{
			Title:       fmt.Sprintf("Created by Dr. %s on %s", doctor.LastName, treatmentPlan.CreationDate.Format(apiservice.TimeFormatLayout)),
			IconURL:     app_url.GetSmallThumbnail(api.DOCTOR_ROLE, doctor.DoctorId.Int64()),
			RoundedIcon: true,
		})

	// TREATMENT VIEWS
	if len(treatmentPlan.TreatmentList.Treatments) > 0 {
		treatmentViews = append(treatmentViews, &tpCardView{
			Views: []tpView{
				&tpTextDisclosureButtonView{
					Style:  captionRegularItalicStyle,
					Text:   "Your prescriptions have been sent to your preferred pharmacy",
					TapURL: app_url.ViewPreferredPharmacyAction(),
				},
			},
		})
		treatmentViews = append(treatmentViews, generateViewsForTreatments(treatmentPlan.TreatmentList, doctor, dataApi, false)...)
	}

	// INSTRUCTION VIEWS
	if treatmentPlan.RegimenPlan != nil && len(treatmentPlan.RegimenPlan.RegimenSections) > 0 {
		cView := &tpCardView{
			Views: []tpView{
				&tpCardTitleView{
					Title:   "Regimen",
					IconURL: app_url.IconRegimen,
				},
			},
		}
		instructionViews = append(instructionViews, cView)

		for i, regimenSection := range treatmentPlan.RegimenPlan.RegimenSections {
			if i > 0 {
				cView.Views = append(cView.Views, &tpSmallDividerView{})
			}
			cView.Views = append(cView.Views, &tpTextView{
				Text:  regimenSection.RegimenName,
				Style: subheaderStyle,
			})

			for i, regimenStep := range regimenSection.RegimenSteps {
				cView.Views = append(cView.Views, &tpListElementView{
					ElementStyle: numberedStyle,
					Number:       i + 1,
					Text:         regimenStep.Text,
				})
			}
		}
	}

	if treatmentPlan.Advice != nil && len(treatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		cView := &tpCardView{
			Views: []tpView{
				&tpCardTitleView{
					Title:       fmt.Sprintf("Dr. %s's Advice", doctor.LastName),
					IconURL:     app_url.GetSmallThumbnail(api.DOCTOR_ROLE, doctor.DoctorId.Int64()),
					RoundedIcon: true,
				},
			},
		}
		instructionViews = append(instructionViews, cView)

		switch len(treatmentPlan.Advice.SelectedAdvicePoints) {
		case 1:
			cView.Views = append(cView.Views, &tpTextView{
				Text: treatmentPlan.Advice.SelectedAdvicePoints[0].Text,
			})
		default:
			for _, advicePoint := range treatmentPlan.Advice.SelectedAdvicePoints {
				cView.Views = append(cView.Views, &tpListElementView{
					ElementStyle: bulletedStyle,
					Text:         advicePoint.Text,
				})
			}
		}
	}

	instructionViews = append(instructionViews, &tpButtonFooterView{
		FooterText: fmt.Sprintf("If you have any questions about your treatment plan, send Dr. %s a message.", doctor.LastName),
		ButtonText: fmt.Sprintf("Message Dr. %s", doctor.LastName),
		IconURL:    app_url.IconMessage,
		TapURL:     app_url.MessageAction(),
	})

	for _, vContainer := range [][]tpView{headerViews, treatmentViews, instructionViews} {
		for _, v := range vContainer {
			if err := v.Validate(); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	}

	apiservice.WriteJSON(w, &treatmentPlanViewsResponse{
		HeaderViews:      headerViews,
		TreatmentViews:   treatmentViews,
		InstructionViews: instructionViews,
	})
}
