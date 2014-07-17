package treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
)

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
}

type treatmentGuideHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentGuideHandler(dataAPI api.DataAPI) *treatmentGuideHandler {
	return &treatmentGuideHandler{dataAPI: dataAPI}
}

func (h *treatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	requestData := new(TreatmentGuideRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	switch apiservice.GetContext(r).Role {
	case api.PATIENT_ROLE:
		h.processTreatmentGuideForPatient(requestData, w, r)
	case api.DOCTOR_ROLE:
		h.processTreatmentGuideForDoctor(requestData, w, r)
	default:
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to determine role from auth token")
	}

}

func (h *treatmentGuideHandler) processTreatmentGuideForPatient(requestData *TreatmentGuideRequestData, w http.ResponseWriter, r *http.Request) {
	patientID, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	}

	treatment, err := h.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get treatment: "+err.Error())
		return
	} else if treatment == nil {
		apiservice.WriteUserError(w, http.StatusNotFound, "Unknown treatment")
		return
	}

	if treatment.PatientId.Int64() != patientID {
		apiservice.WriteUserError(w, http.StatusForbidden, "Patient does not have access to the given treatment")
		return
	}

	treatmentPlan, err := h.dataAPI.GetTreatmentPlanForPatient(treatment.PatientId.Int64(), treatment.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentGuideResponse(h.dataAPI, treatment, treatmentPlan, w, r)
}

func (h *treatmentGuideHandler) processTreatmentGuideForDoctor(requestData *TreatmentGuideRequestData, w http.ResponseWriter, r *http.Request) {
	treatment, err := h.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get treatment: "+err.Error())
		return
	} else if treatment == nil {
		apiservice.WriteUserError(w, http.StatusNotFound, "Unknown treatment")
		return
	}

	treatmentPlan, err := h.dataAPI.GetTreatmentPlanForPatient(treatment.PatientId.Int64(), treatment.TreatmentPlanId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentGuideResponse(h.dataAPI, treatment, treatmentPlan, w, r)
}

func treatmentGuideResponse(dataAPI api.DataAPI, treatment *common.Treatment, treatmentPlan *common.TreatmentPlan, w http.ResponseWriter, r *http.Request) {
	ndc := treatment.DrugDBIds[erx.NDC]
	if ndc == "" {
		apiservice.WriteUserError(w, http.StatusNotFound, "NDC unknown")
		return
	}

	details, err := dataAPI.DrugDetails(ndc)
	if err == api.NoRowsError {
		apiservice.WriteUserError(w, http.StatusNotFound, "No details available")
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get drug details: "+err.Error())
		return
	}

	views, err := treatmentGuideViews(details, treatment, treatmentPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]tpView{"views": views})
}

func treatmentGuideViews(details *common.DrugDetails, treatment *common.Treatment, treatmentPlan *common.TreatmentPlan) ([]tpView, error) {
	var views []tpView

	if details.ImageURL != "" {
		views = append(views,
			&tpImageView{
				ImageURL:    details.ImageURL,
				ImageWidth:  320,
				ImageHeight: 210,
				Insets:      "none",
			},
		)
	}

	views = append(views,
		&tpIconTitleSubtitleView{
			IconURL:  app_url.IconRX,
			Title:    details.Name,
			Subtitle: "", // TODO: Not sure what to put here yet. Possibly details.Alternative.
		},
		&tpSmallDividerView{},
		&tpTextView{
			Style: smallGrayStyle,
			Text:  details.Description,
		},
	)

	if treatment != nil {
		views = append(views,
			&tpLargeDividerView{},
			&tpIconTextView{
				// TODO: This icon info isn't robust or likely accurate
				IconURL:    treatment.Doctor.LargeThumbnailURL,
				IconWidth:  32,
				IconHeight: 32,
				Text:       fmt.Sprintf("%s's Instructions", treatment.Doctor.ShortTitle),
				Style:      sectionHeaderStyle,
			},
			&tpSmallDividerView{},
			&tpTextView{
				Text: treatment.PatientInstructions,
			},
		)
	}

	if len(details.Warnings) != 0 || len(details.Precautions) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "What to Know",
				Style: sectionHeaderStyle,
			},
		)

		if len(details.Warnings) != 0 {
			views = append(views,
				&tpSmallDividerView{},
				&tpTextView{
					Text:  "Warnings",
					Style: subheaderStyle,
				},
			)
			for _, s := range details.Warnings {
				views = append(views, &tpTextView{
					Text:  s,
					Style: "warning",
				})
			}
		}

		if len(details.Precautions) != 0 {
			views = append(views,
				&tpSmallDividerView{},
				&tpTextView{
					Text:  "Precautions",
					Style: subheaderStyle,
				},
			)

			for _, p := range details.Precautions {
				views = append(views, &tpTextView{
					Text: p,
				})
			}
		}
	}

	if len(details.HowToUse) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "How to Use " + details.Name,
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
		)
		for i, s := range details.HowToUse {
			views = append(views, &tpListElementView{
				ElementStyle: "numbered",
				Number:       i + 1,
				Text:         s,
			})
		}
	}

	if len(details.SideEffects) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "Potential Side Effects",
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
		)
		for _, s := range details.SideEffects {
			views = append(views, &tpTextView{
				Text: s,
			})
		}
	}

	if treatment != nil && treatmentPlan != nil {
		views = append(views,
			&tpButtonView{
				Text:    "Message " + treatment.Doctor.ShortTitle,
				IconURL: app_url.IconMessage,
				TapURL:  app_url.SendCaseMessageAction(treatmentPlan.PatientCaseId.Int64()),
			},
		)
	}

	for _, v := range views {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}

	return views, nil
}
