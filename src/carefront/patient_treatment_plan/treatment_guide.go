package patient_treatment_plan

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

type TreatmentGuideRequestData struct {
	TreatmentId int64 `schema:"treatment_id,required"`
}

type PatientTreatmentGuideHandler struct {
	DataAPI api.DataAPI
}

type DoctorTreatmentGuideHandler struct {
	DataAPI api.DataAPI
}

func NewPatientTreatmentGuideHandler(dataAPI api.DataAPI) *PatientTreatmentGuideHandler {
	return &PatientTreatmentGuideHandler{DataAPI: dataAPI}
}

func NewDoctorTreatmentGuideHandler(dataAPI api.DataAPI) *DoctorTreatmentGuideHandler {
	return &DoctorTreatmentGuideHandler{DataAPI: dataAPI}
}

func (h *PatientTreatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(TreatmentGuideRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientID, err := h.DataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get patient: "+err.Error())
		return
	}

	treatment, err := h.DataAPI.GetTreatmentFromId(requestData.TreatmentId)
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

	treatmentGuideResponse(h.DataAPI, w, treatment)
}

func (h *DoctorTreatmentGuideHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(TreatmentGuideRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorID, err := h.DataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get doctor: "+err.Error())
		return
	}

	treatment, err := h.DataAPI.GetTreatmentFromId(requestData.TreatmentId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get treatment: "+err.Error())
		return
	} else if treatment == nil {
		apiservice.WriteUserError(w, http.StatusNotFound, "Unknown treatment")
		return
	}

	if err := apiservice.VerifyDoctorPatientRelationship(h.DataAPI, treatment.Doctor, treatment.Patient); err != nil {
		golog.Warningf("Doctor %d does not have access to treatment %d: %s", doctorID, treatment.Id.Int64(), err.Error())
		apiservice.WriteUserError(w, http.StatusForbidden, "Doctor does not have access to the given treatment")
		return
	}

	treatmentGuideResponse(h.DataAPI, w, treatment)
}

func treatmentGuideResponse(dataAPI api.DataAPI, w http.ResponseWriter, treatment *common.Treatment) {
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

	// Format drug details into views

	views := []TPView{
		&TPIconTitleSubtitleView{
			IconURL:  "spruce:///images/icon_rx",
			Title:    details.Name,
			Subtitle: details.Subtitle,
		},
		&TPSmallDividerView{},
		&TPTextView{
			Style: smallGrayStyle,
			Text:  details.Description,
		},
		&TPLargeDividerView{},
		&TPIconTextView{
			// TODO: This icon info isn't robust or likely accurate
			IconURL:    fmt.Sprintf("spruce:///images/doctor_photo_%d", treatment.Doctor.DoctorId.Int64()),
			IconWidth:  32,
			IconHeight: 32,
			Text:       fmt.Sprintf("Dr. %s's Instructions", treatment.Doctor.LastName),
			Style:      sectionHeaderStyle,
		},
		&TPSmallDividerView{},
		&TPTextView{
			Text: treatment.PatientInstructions,
		},
		&TPLargeDividerView{},
		&TPTextView{
			Text:  "What to Know",
			Style: sectionHeaderStyle,
		},
		&TPSmallDividerView{},
	}

	if len(details.Warnings) != 0 {
		views = append(views, &TPTextView{
			Text:  "Warnings",
			Style: subheaderStyle,
		})
		for _, s := range details.Warnings {
			views = append(views, &TPTextView{
				Text:  s,
				Style: "warning",
			})
		}
	}

	if len(details.Precautions) != 0 {
		views = append(views, &TPTextView{
			Text:  "Precautions",
			Style: subheaderStyle,
		})
		for _, p := range details.Precautions {
			views = append(views, &TPSnippetDetailsView{
				Snippet: p.Snippet,
				Details: p.Details,
			})
		}
	}

	views = append(views,
		&TPLargeDividerView{},
		&TPTextView{
			Text:  "How to Use " + details.Name,
			Style: sectionHeaderStyle,
		},
		&TPSmallDividerView{},
	)

	for i, s := range details.HowToUse {
		views = append(views, &TPListElementView{
			ElementStyle: "numbered",
			Number:       i + 1,
			Text:         s,
		})
	}

	if len(details.DoNots) != 0 {
		views = append(views, &TPSmallDividerView{})
		for _, s := range details.DoNots {
			views = append(views, &TPListElementView{
				ElementStyle: "dont",
				Text:         s,
			})
		}
	}

	views = append(views,
		&TPLargeDividerView{},
		&TPTextView{
			Text:  "Message Your Doctor If\u2026",
			Style: sectionHeaderStyle,
		},
		&TPSmallDividerView{},
	)

	for _, s := range details.MessageDoctorIf {
		views = append(views, &TPTextView{
			Text: s,
		})
	}

	views = append(views,
		&TPPlainButtonView{
			Text:   fmt.Sprintf("View all %s side effects", details.Name),
			TapURL: "spruce:///action/view_side_effects",
		},
		&TPButtonView{
			Text:    "Message Dr. " + treatment.Doctor.LastName,
			IconURL: "spruce:///images/icon_message",
			TapURL:  "spruce:///action/message_doctor",
		},
	)

	for _, v := range views {
		if err := v.Validate(); err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to render views: "+err.Error())
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, map[string][]TPView{"views": views})
}
