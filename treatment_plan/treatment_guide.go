package treatment_plan

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
)

var footerText = `This prescription guide covers only common use and is not meant to be a complete listing of drug information. If you are experiencing concerning symptoms, seek medical attention immediately.

For more information, please see the package insert that came with your medication or ask your pharmacist or physician directly.`

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
}

type treatmentGuideHandler struct {
	dataAPI api.DataAPI
}

func NewTreatmentGuideHandler(dataAPI api.DataAPI) http.Handler {
	return &treatmentGuideHandler{dataAPI: dataAPI}
}

func (h *treatmentGuideHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

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

	treatmentPlan, err := h.dataAPI.GetTreatmentPlanForPatient(treatment.PatientId.Int64(), treatment.TreatmentPlanID.Int64())
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
		ctxt.RequestCache[apiservice.PatientID] = patientID

		if treatment.PatientId.Int64() != patientID {
			return false, apiservice.NewAccessForbiddenError()
		}

	case api.DOCTOR_ROLE:
		doctorID, err := h.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.DoctorID] = doctorID

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

	name := details.Name
	if treatment != nil {
		name = fmt.Sprintf("%s %s %s", treatment.DrugName, treatment.DosageStrength, treatment.DrugForm)
	}
	views = append(views,
		&tpIconTitleSubtitleView{
			Title:    name,
			Subtitle: details.OtherNames,
		},
		&tpSmallDividerView{},
		&tpTextView{
			Text: details.Description,
		},
	)

	if treatment != nil || len(details.Tips) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "Instructions",
				Style: sectionHeaderStyle,
			},
		)

		if treatment != nil {
			views = append(views,
				&tpSmallDividerView{},
				&tpTextView{
					Text:  strings.ToUpper(fmt.Sprintf("%s's Instructions", treatment.Doctor.ShortDisplayName)),
					Style: subheaderStyle,
				},
				&tpTextView{
					Text: treatment.PatientInstructions,
				},
			)
		}

		if len(details.Tips) != 0 {
			views = append(views,
				&tpSmallDividerView{},
				&tpTextView{
					Text:  "TIPS",
					Style: subheaderStyle,
				},
			)
			for _, t := range details.Tips {
				views = append(views,
					&tpTextView{
						Text: t,
					},
				)
			}
		}
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
				Text: s,
			})
		}
	}

	if len(details.CommonSideEffects) != 0 {
		views = append(views,
			&tpLargeDividerView{},
			&tpTextView{
				Text:  "Common Side Effects",
				Style: sectionHeaderStyle,
			},
			&tpSmallDividerView{},
		)
		for _, s := range details.CommonSideEffects {
			views = append(views, &tpTextView{
				Text: s,
			})
		}
	}

	if treatment != nil && treatmentPlan != nil {
		views = append(views,
			&tpButtonFooterView{
				FooterText: footerText,
				ButtonText: "Message Care Team",
				IconURL:    app_url.IconMessage,
				TapURL:     app_url.SendCaseMessageAction(treatmentPlan.PatientCaseId.Int64()),
			},
		)
	} else {
		views = append(views,
			&tpButtonFooterView{
				FooterText: footerText,
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
