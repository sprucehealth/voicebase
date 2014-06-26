package layout

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/info_intake"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/SpruceHealth/mapstructure"
)

const (
	layoutSyntaxVersion = 1
)

type layoutUploadHandler struct {
	dataAPI api.DataAPI
}

func NewLayoutUploadHandler(dataAPI api.DataAPI) *layoutUploadHandler {
	return &layoutUploadHandler{
		dataAPI: dataAPI,
	}
}

func (h *layoutUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	if apiservice.GetContext(r).Role != api.ADMIN_ROLE {
		apiservice.WriteAccessNotAllowedError(w, r)
		return
	}

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	layouts := map[string][]byte{
		"intake":   nil,
		"review":   nil,
		"diagnose": nil,
	}

	// Read the uploaded layouts and get health condition tag

	var healthCondition string

	newCount := 0
	for name := range layouts {
		if file, _, err := r.FormFile(name); err != http.ErrMissingFile {
			if err != nil {
				apiservice.WriteBadRequestError(err, w, r)
				return
			}
			layouts[name], err = ioutil.ReadAll(file)
			if err != nil {
				apiservice.WriteBadRequestError(err, w, r)
				return
			}

			// Parse the json to get the health condition which is needed to fetch
			// active templates.

			var js map[string]interface{}
			if err = json.Unmarshal(layouts[name], &js); err != nil {
				apiservice.WriteValidationError("Failed to parse json: "+err.Error(), w, r)
				return
			}
			var condition string
			if v, ok := js["health_condition"]; ok {
				switch x := v.(type) {
				case string: // patient intake and doctor review
					condition = x
				case map[string]interface{}: // diagnosis has it at the second level
					if c, ok := x["health_condition"].(string); ok {
						condition = c
					}
				}
			}
			if condition == "" {
				apiservice.WriteValidationError("health_condition is not set", w, r)
				return
			}

			if healthCondition == "" {
				healthCondition = condition
			} else if healthCondition != condition {
				apiservice.WriteValidationError("Health condition for all layouts must match", w, r)
				return
			}

			newCount++
		}
	}

	if newCount == 0 {
		apiservice.WriteBadRequestError(errors.New("no layouts attached"), w, r)
		return
	}

	// Parse the layouts and get active layout for anything not uploaded

	var intakeLayout *info_intake.InfoIntakeLayout
	var reviewLayout *info_intake.DVisitReviewSectionListView
	var reviewJS map[string]interface{}
	var diagnoseLayout *info_intake.DiagnosisIntake

	conditionID, err := h.dataAPI.GetHealthConditionInfo(healthCondition)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Patient intake

	data := layouts["intake"]
	if data == nil {
		data, _, err = h.dataAPI.GetCurrentActivePatientLayout(api.EN_LANGUAGE_ID, conditionID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if err = json.Unmarshal(data, &intakeLayout); err != nil {
		apiservice.WriteValidationError("Failed to parse json: "+err.Error(), w, r)
		return
	}

	// Doctor review

	data = layouts["review"]
	if data == nil {
		data, _, err = h.dataAPI.GetCurrentActiveDoctorLayout(conditionID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if err := json.Unmarshal(data, &reviewJS); err != nil {
		apiservice.WriteValidationError("Failed to parse json: "+err.Error(), w, r)
		return
	}
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:   &reviewLayout,
		TagName:  "json",
		Registry: *info_intake.DVisitReviewViewTypeRegistry,
	})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := d.Decode(reviewJS["visit_review"]); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Doctor diagnose

	data = layouts["diagnose"]
	if data != nil {
		if err = json.Unmarshal(data, &diagnoseLayout); err != nil {
			apiservice.WriteValidationError("Failed to parse json: "+err.Error(), w, r)
			return
		}
	}

	// Validate layouts

	if err := api.FillIntakeLayout(intakeLayout, h.dataAPI, api.EN_LANGUAGE_ID); err != nil {
		// TODO: this could be a validation error (unknown question or answer) or an internal error.
		// There's currently no easy way to tell the difference. This is ok for now since this is
		// an admin endpoint.
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if err := validatePatientLayout(intakeLayout); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if err := compareQuestions(intakeLayout, reviewJS); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if diagnoseLayout != nil {
		if err := api.FillDiagnosisIntake(diagnoseLayout, h.dataAPI, api.EN_LANGUAGE_ID); err != nil {
			// TODO: this could be a validation error (unknown question or answer) or an internal error.
			// There's currently no easy way to tell the difference. This is ok for now since this is
			// an admin endpoint.
			apiservice.WriteValidationError(err.Error(), w, r)
			return
		}
	}

	// Make sure the review layout renders

	context, err := reviewContext(intakeLayout)
	if err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if _, err = reviewLayout.Render(context); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// The layouts should now be considered valid (hopefully). So, save
	// any that were uploaded and ignore the ones that were pulled from
	// the database.

	var intakeModelID int64
	var intakeModelVersionIDs []int64
	var reviewModelID, reviewLayoutID int64
	var diagnoseModelID, diagnoseLayoutID int64

	if data := layouts["intake"]; data != nil {
		intakeModelID, err = h.dataAPI.CreateLayoutVersion(data, layoutSyntaxVersion, conditionID,
			api.PATIENT_ROLE, api.CONDITION_INTAKE_PURPOSE, "automatically generated")
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// get all the supported languages
		_, supportedLanguageIDs, err := h.dataAPI.GetSupportedLanguages()
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err = json.Unmarshal(data, &intakeLayout); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// generate a client layout for each language
		intakeModelVersionIDs = make([]int64, len(supportedLanguageIDs))
		for i, supportedLanguageID := range supportedLanguageIDs {
			clientModel := intakeLayout
			if err := api.FillIntakeLayout(clientModel, h.dataAPI, supportedLanguageID); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			jsonData, err := json.Marshal(&clientModel)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			// mark the client layout as creating until we have uploaded all client layouts before marking it as ACTIVE
			intakeModelVersionIDs[i], err = h.dataAPI.CreatePatientLayout(jsonData, supportedLanguageID, intakeModelID, clientModel.HealthConditionId)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	}

	if data := layouts["review"]; data != nil {
		reviewModelID, err = h.dataAPI.CreateLayoutVersion(data, layoutSyntaxVersion, conditionID,
			api.DOCTOR_ROLE, api.REVIEW_PURPOSE, "automatically generated")
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// Remarshal to compact the JSON
		data, err = json.Marshal(reviewJS)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		reviewLayoutID, err = h.dataAPI.CreateDoctorLayout(data, reviewModelID, conditionID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if data := layouts["diagnose"]; data != nil {
		diagnoseModelID, err = h.dataAPI.CreateLayoutVersion(data, layoutSyntaxVersion, conditionID,
			api.DOCTOR_ROLE, api.DIAGNOSE_PURPOSE, "automatically generated")
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// Remarshal now that the layout is filled in (which was done above during validation)
		data, err = json.Marshal(diagnoseLayout)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		diagnoseLayoutID, err = h.dataAPI.CreateDoctorLayout(data, diagnoseModelID, conditionID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// Make all layouts active. This is done last to lessen the chance of inconsistent
	// layouts being active if one of the creates fails since there's no global
	// transaction.
	if intakeModelID != 0 {
		if err := h.dataAPI.UpdatePatientActiveLayouts(intakeModelID, intakeModelVersionIDs, conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if reviewModelID != 0 {
		if err := h.dataAPI.UpdateDoctorActiveLayouts(reviewModelID, reviewLayoutID, conditionID, api.REVIEW_PURPOSE); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if diagnoseModelID != 0 {
		if err := h.dataAPI.UpdateDoctorActiveLayouts(diagnoseModelID, diagnoseLayoutID, conditionID, api.DIAGNOSE_PURPOSE); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
