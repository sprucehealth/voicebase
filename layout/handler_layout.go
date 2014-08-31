package layout

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

const (
	layoutSyntaxVersion = 1
)

type layoutUploadHandler struct {
	dataAPI api.DataAPI
}

func NewLayoutUploadHandler(dataAPI api.DataAPI) http.Handler {
	return &layoutUploadHandler{
		dataAPI: dataAPI,
	}
}

func (h *layoutUploadHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	if apiservice.GetContext(r).Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}
	return true, nil
}

type layoutInfo struct {
	Data        []byte
	FileName    string
	Version     *common.Version
	UpgradeType common.VersionComponent
}

func (h *layoutUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		apiservice.WriteBadRequestError(err, w, r)
		return
	}

	rData := &requestData{}

	err := rData.populateTemplatesAndHealthCondition(r, h.dataAPI)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// validate the intake/review pairing based on what layouts are being uploaded and versioned
	if err := rData.validateUpgradePathsAndLayouts(r, h.dataAPI); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// parse and validate diagnosis layout
	if err := rData.parseAndValidateDiagnosisLayout(r, h.dataAPI); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// The layouts should now be considered valid (hopefully). So, save
	// any that were uploaded and ignore the ones that were pulled from
	// the database.
	var intakeModelID int64
	var intakeModelVersionIDs []int64
	var reviewModelID, reviewLayoutID int64
	var diagnoseModelID, diagnoseLayoutID int64

	if rData.intakeLayoutInfo != nil {
		layout := &api.LayoutTemplateVersion{
			Role:              api.PATIENT_ROLE,
			Purpose:           api.ConditionIntakePurpose,
			Version:           *rData.intakeLayoutInfo.Version,
			Layout:            rData.intakeLayoutInfo.Data,
			HealthConditionID: rData.conditionID,
			Status:            api.STATUS_CREATING,
		}
		err := h.dataAPI.CreateLayoutTemplateVersion(layout)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		intakeModelID = layout.ID

		// get all the supported languages
		_, supportedLanguageIDs, err := h.dataAPI.GetSupportedLanguages()
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// generate a client layout for each language
		intakeModelVersionIDs = make([]int64, len(supportedLanguageIDs))
		for i, supportedLanguageID := range supportedLanguageIDs {
			clientModel := rData.intakeLayout
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
			filledIntakeLayout := &api.LayoutVersion{
				Purpose:                 api.ConditionIntakePurpose,
				Version:                 *rData.intakeLayoutInfo.Version,
				Layout:                  jsonData,
				LayoutTemplateVersionID: intakeModelID,
				HealthConditionID:       rData.conditionID,
				LanguageID:              supportedLanguageID,
				Status:                  api.STATUS_CREATING,
			}
			if err := h.dataAPI.CreateLayoutVersion(filledIntakeLayout); err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
			intakeModelVersionIDs[i] = filledIntakeLayout.ID
		}
	}

	if rData.reviewLayoutInfo != nil {
		layoutTemplate := &api.LayoutTemplateVersion{
			Role:              api.DOCTOR_ROLE,
			Purpose:           api.ReviewPurpose,
			Version:           *rData.reviewLayoutInfo.Version,
			Layout:            rData.reviewLayoutInfo.Data,
			HealthConditionID: rData.conditionID,
			Status:            api.STATUS_CREATING,
		}

		if err := h.dataAPI.CreateLayoutTemplateVersion(layoutTemplate); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		reviewModelID = layoutTemplate.ID

		// Remarshal to compact the JSON
		data, err := json.Marshal(rData.reviewJS)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		filledReviewLayout := &api.LayoutVersion{
			Purpose:                 api.ReviewPurpose,
			Version:                 *rData.reviewLayoutInfo.Version,
			Layout:                  data,
			LayoutTemplateVersionID: reviewModelID,
			HealthConditionID:       rData.conditionID,
			LanguageID:              api.EN_LANGUAGE_ID,
			Status:                  api.STATUS_CREATING,
		}

		if err := h.dataAPI.CreateLayoutVersion(filledReviewLayout); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		reviewLayoutID = filledReviewLayout.ID
	}

	if rData.diagnoseLayoutInfo != nil {
		diagnoseTemplate := &api.LayoutTemplateVersion{
			Role:              api.DOCTOR_ROLE,
			Purpose:           api.DiagnosePurpose,
			Version:           *rData.diagnoseLayoutInfo.Version,
			Layout:            rData.diagnoseLayoutInfo.Data,
			HealthConditionID: rData.conditionID,
			Status:            api.STATUS_CREATING,
		}

		if err := h.dataAPI.CreateLayoutTemplateVersion(diagnoseTemplate); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		diagnoseModelID = diagnoseTemplate.ID

		// Remarshal now that the layout is filled in (which was done during validation)
		data, err := json.Marshal(rData.diagnoseLayout)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		filledDiagnoseLayout := &api.LayoutVersion{
			Purpose:                 api.DiagnosePurpose,
			Version:                 *rData.diagnoseLayoutInfo.Version,
			Layout:                  data,
			LayoutTemplateVersionID: diagnoseModelID,
			HealthConditionID:       rData.conditionID,
			LanguageID:              api.EN_LANGUAGE_ID,
			Status:                  api.STATUS_CREATING,
		}

		if err := h.dataAPI.CreateLayoutVersion(filledDiagnoseLayout); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		diagnoseLayoutID = filledDiagnoseLayout.ID
	}

	// Make all layouts active. This is done last to lessen the chance of inconsistent
	// layouts being active if one of the creates fails since there's no global
	// transaction.
	if intakeModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.ConditionIntakePurpose, rData.intakeLayoutInfo.Version, intakeModelID, intakeModelVersionIDs, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if reviewModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.ReviewPurpose, rData.reviewLayoutInfo.Version, reviewModelID, []int64{reviewLayoutID}, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	if diagnoseModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.DiagnosePurpose, rData.diagnoseLayoutInfo.Version, diagnoseModelID, []int64{diagnoseLayoutID}, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// Now that the layouts have been successfully created
	// create any new mappings for the layouts

	if rData.intakeUpgradeType == common.Major {
		if err := h.dataAPI.CreateAppVersionMapping(rData.patientAppVersion, rData.platform, rData.intakeLayoutInfo.Version.Major, api.PATIENT_ROLE, api.ConditionIntakePurpose, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if rData.reviewUpgradeType == common.Major {
		if err := h.dataAPI.CreateAppVersionMapping(rData.doctorAppVersion, rData.platform, rData.reviewLayoutInfo.Version.Major, api.DOCTOR_ROLE, api.ReviewPurpose, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	if rData.reviewUpgradeType == common.Major || rData.reviewUpgradeType == common.Minor {
		if err := h.dataAPI.CreateLayoutMapping(rData.intakeLayoutInfo.Version.Major, rData.intakeLayoutInfo.Version.Minor,
			rData.reviewLayoutInfo.Version.Major, rData.reviewLayoutInfo.Version.Minor, rData.conditionID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var message string
	if rData.intakeLayoutInfo != nil {
		message += "Intake Layout underwent a " + string(rData.intakeUpgradeType) + " version upgrade\n"
	}
	if rData.reviewLayoutInfo != nil {
		message += "Review Layout underwent a " + string(rData.reviewUpgradeType) + " version upgrade\n"
	}

	apiservice.WriteJSON(w, map[string]interface{}{
		"message": message,
	})
}
