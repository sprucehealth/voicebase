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

func NewTreatmentGuideHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&treatmentGuideHandler{dataAPI: dataAPI}, []string{apiservice.HTTP_GET})
}

func (h *treatmentGuideHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := new(TreatmentGuideRequestData)
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	treatment, err := h.dataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		return false, err
	} else if treatment == nil {
		return false, apiservice.NewResourceNotFoundError("treatment not found", r)
	}
	ctxt.RequestCache[apiservice.Treatment] = treatment

	treatmentPlan, err := h.dataAPI.GetTreatmentPlanForPatient(treatment.PatientId.Int64(), treatment.TreatmentPlanId.Int64())
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

	switch ctxt.Role {
	case api.PATIENT_ROLE:
		patientID, err := h.dataAPI.GetPatientIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientId] = patientID

		if treatment.PatientId.Int64() != patientID {
			return false, apiservice.NewAccessForbiddenError()
		}

	case api.DOCTOR_ROLE:
		doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.DoctorId] = doctorID

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID, treatmentPlan.PatientId.Int64(), treatmentPlan.PatientCaseId.Int64(), h.dataAPI); err != nil {
			return false, err
		}

		// ensure that doctor is owner of the treatment plan
		if doctorID != treatmentPlan.DoctorId.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}
	}

	return true, nil
}

func (h *treatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	treatment := ctxt.RequestCache[apiservice.Treatment].(*common.Treatment)
	treatmentPlan := ctxt.RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

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

	if len(details.Warnings) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "Warnings",
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
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
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "Precautions",
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
		)

		for _, p := range details.Precautions {
			views = append(views, &tpTextView{
				Text: p,
			})
		}
	}

	if len(details.HowToUse) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "How to Use",
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
		)
		for _, s := range details.HowToUse {
			views = append(views, &tpTextView{
				Text: s,
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
