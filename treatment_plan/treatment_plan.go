package treatment_plan

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
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
	} else if requestData.TreatmentPlanId == 0 {
		apiservice.WriteValidationError("treatment_plan_id must be specified", w, r)
		return
	}

	var doctor *common.Doctor
	var patient *common.Patient
	var treatmentPlan *common.TreatmentPlan
	var err error
	switch apiservice.GetContext(r).Role {
	case api.PATIENT_ROLE:
		patient, err = p.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
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

		doctor, err = p.dataApi.GetDoctorFromId(treatmentPlan.DoctorId.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		treatmentPlanResponse(p.dataApi, w, r, treatmentPlan, doctor, patient)

	case api.DOCTOR_ROLE:
		doctor, err = p.dataApi.GetDoctorFromAccountId(apiservice.GetContext(r).AccountId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		patient, err = p.dataApi.GetPatientFromTreatmentPlanId(requestData.TreatmentPlanId)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err = apiservice.ValidateDoctorAccessToPatientFile(doctor.DoctorId.Int64(), patient.PatientId.Int64(), p.dataApi); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		treatmentPlan = &common.TreatmentPlan{
			Id:       encoding.NewObjectId(requestData.TreatmentPlanId),
			DoctorId: encoding.NewObjectId(doctor.DoctorId.Int64()),
		}

	default:
		apiservice.WriteValidationError("Unable to identify role", w, r)
	}

	err = populateTreatmentPlan(p.dataApi, treatmentPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	treatmentPlanResponse(p.dataApi, w, r, treatmentPlan, doctor, patient)
}

func treatmentPlanResponse(dataApi api.DataAPI, w http.ResponseWriter, r *http.Request, treatmentPlan *common.TreatmentPlan, doctor *common.Doctor, patient *common.Patient) {
	views := make([]TPView, 0)
	views = append(views, &TPVisitHeaderView{
		ImageURL: doctor.LargeThumbnailUrl,
		Title:    fmt.Sprintf("Dr. %s %s", doctor.FirstName, doctor.LastName),
		Subtitle: "Dermatologist",
	})

	views = append(views, &TPImageView{
		ImageWidth:  125,
		ImageHeight: 45,
		ImageURL:    app_url.TmpSignature.String(),
	})

	views = append(views, &TPLargeDividerView{})

	if len(treatmentPlan.TreatmentList.Treatments) > 0 {
		views = append(views, &TPTextView{
			Text:  "Prescriptions",
			Style: "section_header",
		})

		for _, treatment := range treatmentPlan.TreatmentList.Treatments {
			views = append(views, &TPSmallDividerView{})

			iconURL := app_url.IconRX
			if treatment.OTC {
				iconURL = app_url.IconOTC
			}

			// only include tapurl and buttontitle if drug details
			// exist
			var buttonTitle string
			var tapUrl *app_url.SpruceAction
			if ndc := treatment.DrugDBIds[erx.NDC]; ndc != "" {
				if exists, err := dataApi.DoesDrugDetailsExist(ndc); exists {
					buttonTitle = "What to know about " + treatment.DrugName
					tapUrl = app_url.ViewTreatmentGuideAction(treatment.Id.Int64())
				} else if err != nil && err != api.NoRowsError {
					golog.Errorf("Error when trying to check if drug details exist: %s", err)
				}
			}

			views = append(views, &TPPrescriptionView{
				IconURL:     iconURL,
				Title:       fmt.Sprintf("%s %s", treatment.DrugInternalName, treatment.DosageStrength),
				Description: treatment.PatientInstructions,
				ButtonTitle: buttonTitle,
				TapURL:      tapUrl,
			})
		}
	}

	if treatmentPlan.RegimenPlan != nil && len(treatmentPlan.RegimenPlan.RegimenSections) > 0 {
		views = append(views, &TPLargeDividerView{})
		views = append(views, &TPTextView{
			Text:  "Personal Regimen",
			Style: sectionHeaderStyle,
		})

		for _, regimenSection := range treatmentPlan.RegimenPlan.RegimenSections {
			views = append(views, &TPSmallDividerView{})
			views = append(views, &TPTextView{
				Text:  regimenSection.RegimenName,
				Style: subheaderStyle,
			})

			for i, regimenStep := range regimenSection.RegimenSteps {
				views = append(views, &TPListElementView{
					ElementStyle: "numbered",
					Number:       i + 1,
					Text:         regimenStep.Text,
				})
			}
		}
	}

	if treatmentPlan.Advice != nil && len(treatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		views = append(views, &TPLargeDividerView{})
		views = append(views, &TPTextView{
			Text:  fmt.Sprintf("Dr. %s's Advice", doctor.LastName),
			Style: sectionHeaderStyle,
		})

		switch len(treatmentPlan.Advice.SelectedAdvicePoints) {
		case 1:
			views = append(views, &TPTextView{
				Text: treatmentPlan.Advice.SelectedAdvicePoints[0].Text,
			})
		default:
			for _, advicePoint := range treatmentPlan.Advice.SelectedAdvicePoints {
				views = append(views, &TPListElementView{
					ElementStyle: "buletted",
					Text:         advicePoint.Text,
				})
			}
		}
	}

	views = append(views, &TPLargeDividerView{})
	views = append(views, &TPTextView{
		Text:  "Next Steps",
		Style: sectionHeaderStyle,
	})

	views = append(views, &TPSmallDividerView{})
	views = append(views, &TPTextView{
		Text: "Your prescriptions have been sent to your pharmacy and will be ready for pick soon.",
	})

	if patient.Pharmacy != nil {
		views = append(views, &TPSmallDividerView{})
		views = append(views, &TPTextView{
			Text:  "Your Pharmacy",
			Style: subheaderStyle,
		})

		views = append(views, &TPPharmacyMapView{
			Pharmacy: patient.Pharmacy,
		})
	}

	// identify prescriptions to pickup
	rxTreatments := make([]*common.Treatment, 0, len(treatmentPlan.TreatmentList.Treatments))
	for _, treatment := range treatmentPlan.TreatmentList.Treatments {
		if !treatment.OTC {
			rxTreatments = append(rxTreatments, treatment)
		}
	}

	if len(rxTreatments) > 0 {
		views = append(views, &TPSmallDividerView{})
		views = append(views, &TPTextView{
			Text:  "Prescriptions to Pick Up",
			Style: subheaderStyle,
		})

		treatmentListView := &TPTreatmentListView{}
		treatmentListView.Treatments = make([]*TPIconTextView, len(rxTreatments))
		for i, rxTreatment := range rxTreatments {
			treatmentListView.Treatments[i] = &TPIconTextView{
				IconURL:   app_url.IconRX,
				Text:      fmt.Sprintf("%s %s", rxTreatment.DrugInternalName, rxTreatment.DosageStrength),
				TextStyle: "bold",
			}
		}
		views = append(views, treatmentListView)
	}

	views = append(views, &TPButtonFooterView{
		FooterText: fmt.Sprintf("If you have any questions or concerns regarding your treatment plan, send Dr. %s a message.", doctor.LastName),
		ButtonText: fmt.Sprintf("Message Dr. %s", doctor.LastName),
		IconURL:    app_url.IconMessage,
		TapURL:     app_url.MessageAction(),
	})

	for _, v := range views {
		if err := v.Validate(); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to render views: "+err.Error())
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]tpView{"views": views})
}
