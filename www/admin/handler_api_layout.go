package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

const (
	maxMemoryUsage = 2 * 1024 * 1024 // MB
	intake         = "intake"
	review         = "review"
	diagnose       = "diagnose"
)

type layoutUploadHandler struct {
	dataAPI api.DataAPI
}

func NewLayoutUploadHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&layoutUploadHandler{dataAPI: dataAPI}, httputil.Post)
}

type layoutInfo struct {
	Data        []byte
	FileName    string
	Version     *common.Version
	UpgradeType common.VersionComponent
}

func (h *layoutUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMemoryUsage); err != nil {
		www.APIBadRequestError(w, r, "Failed to parse form.")
		return
	}

	rData := &requestData{}

	err := rData.populateTemplatesAndPathway(r, h.dataAPI)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// validate the intake/review pairing based on what layouts are being uploaded and versioned
	if err := rData.validateUpgradePathsAndLayouts(r, h.dataAPI); err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	// parse and validate diagnosis layout
	if err := rData.parseAndValidateDiagnosisLayout(r, h.dataAPI); err != nil {
		www.APIBadRequestError(w, r, err.Error())
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
			Role:      api.RolePatient,
			Purpose:   api.ConditionIntakePurpose,
			Version:   *rData.intakeLayoutInfo.Version,
			Layout:    rData.intakeLayoutInfo.Data,
			PathwayID: rData.pathwayID,
			Status:    api.StatusCreating,
			SKUID:     rData.skuID,
		}
		err := h.dataAPI.CreateLayoutTemplateVersion(layout)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		intakeModelID = layout.ID

		// get all the supported languages
		_, supportedLanguageIDs, err := h.dataAPI.GetSupportedLanguages()
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		// generate a client layout for each language
		intakeModelVersionIDs = make([]int64, len(supportedLanguageIDs))
		for i, supportedLanguageID := range supportedLanguageIDs {
			clientModel := rData.intakeLayout
			if err := api.FillIntakeLayout(clientModel, h.dataAPI, supportedLanguageID); err != nil {
				www.APIInternalError(w, r, err)
				return
			}

			jsonData, err := json.Marshal(&clientModel)
			if err != nil {
				www.APIInternalError(w, r, err)
				return
			}

			// mark the client layout as creating until we have uploaded all client layouts before marking it as ACTIVE
			filledIntakeLayout := &api.LayoutVersion{
				Purpose:                 api.ConditionIntakePurpose,
				Version:                 *rData.intakeLayoutInfo.Version,
				Layout:                  jsonData,
				LayoutTemplateVersionID: intakeModelID,
				PathwayID:               rData.pathwayID,
				LanguageID:              supportedLanguageID,
				Status:                  api.StatusCreating,
				SKUID:                   rData.skuID,
			}
			if err := h.dataAPI.CreateLayoutVersion(filledIntakeLayout); err != nil {
				www.APIInternalError(w, r, err)
				return
			}
			intakeModelVersionIDs[i] = filledIntakeLayout.ID
		}
	}

	if rData.reviewLayoutInfo != nil {
		layoutTemplate := &api.LayoutTemplateVersion{
			Role:      api.RoleDoctor,
			Purpose:   api.ReviewPurpose,
			Version:   *rData.reviewLayoutInfo.Version,
			Layout:    rData.reviewLayoutInfo.Data,
			PathwayID: rData.pathwayID,
			Status:    api.StatusCreating,
			SKUID:     rData.skuID,
		}

		if err := h.dataAPI.CreateLayoutTemplateVersion(layoutTemplate); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		reviewModelID = layoutTemplate.ID

		// Remarshal to compact the JSON
		data, err := json.Marshal(rData.reviewJS)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		filledReviewLayout := &api.LayoutVersion{
			Purpose:                 api.ReviewPurpose,
			Version:                 *rData.reviewLayoutInfo.Version,
			Layout:                  data,
			LayoutTemplateVersionID: reviewModelID,
			PathwayID:               rData.pathwayID,
			LanguageID:              api.LanguageIDEnglish,
			Status:                  api.StatusCreating,
			SKUID:                   rData.skuID,
		}

		if err := h.dataAPI.CreateLayoutVersion(filledReviewLayout); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		reviewLayoutID = filledReviewLayout.ID
	}

	if rData.diagnoseLayoutInfo != nil {
		diagnoseTemplate := &api.LayoutTemplateVersion{
			Role:      api.RoleDoctor,
			Purpose:   api.DiagnosePurpose,
			Version:   *rData.diagnoseLayoutInfo.Version,
			Layout:    rData.diagnoseLayoutInfo.Data,
			PathwayID: rData.pathwayID,
			Status:    api.StatusCreating,
		}

		if err := h.dataAPI.CreateLayoutTemplateVersion(diagnoseTemplate); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		diagnoseModelID = diagnoseTemplate.ID

		// Remarshal now that the layout is filled in (which was done during validation)
		data, err := json.Marshal(rData.diagnoseLayout)
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		filledDiagnoseLayout := &api.LayoutVersion{
			Purpose:                 api.DiagnosePurpose,
			Version:                 *rData.diagnoseLayoutInfo.Version,
			Layout:                  data,
			LayoutTemplateVersionID: diagnoseModelID,
			PathwayID:               rData.pathwayID,
			LanguageID:              api.LanguageIDEnglish,
			Status:                  api.StatusCreating,
		}

		if err := h.dataAPI.CreateLayoutVersion(filledDiagnoseLayout); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
		diagnoseLayoutID = filledDiagnoseLayout.ID
	}

	// Make all layouts active. This is done last to lessen the chance of inconsistent
	// layouts being active if one of the creates fails since there's no global
	// transaction.
	if intakeModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.ConditionIntakePurpose, rData.intakeLayoutInfo.Version, intakeModelID, intakeModelVersionIDs, rData.pathwayID, rData.skuID); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}
	if reviewModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.ReviewPurpose, rData.reviewLayoutInfo.Version, reviewModelID, []int64{reviewLayoutID}, rData.pathwayID, rData.skuID); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}
	if diagnoseModelID != 0 {
		if err := h.dataAPI.UpdateActiveLayouts(api.DiagnosePurpose, rData.diagnoseLayoutInfo.Version, diagnoseModelID, []int64{diagnoseLayoutID}, rData.pathwayID, nil); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

	// Now that the layouts have been successfully created
	// create any new mappings for the layouts

	if rData.intakeUpgradeType == common.Major {
		if err := h.dataAPI.CreateAppVersionMapping(rData.patientAppVersion, rData.platform, rData.intakeLayoutInfo.Version.Major, api.RolePatient, api.ConditionIntakePurpose, rData.pathwayID, rData.skuType); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

	if rData.reviewUpgradeType == common.Major {
		if err := h.dataAPI.CreateAppVersionMapping(rData.doctorAppVersion, rData.platform, rData.reviewLayoutInfo.Version.Major, api.RoleDoctor, api.ReviewPurpose, rData.pathwayID, rData.skuType); err != nil {
			www.APIInternalError(w, r, err)
			return
		}
	}

	if rData.reviewUpgradeType == common.Major || rData.reviewUpgradeType == common.Minor {
		if err := h.dataAPI.CreateLayoutMapping(rData.intakeLayoutInfo.Version.Major, rData.intakeLayoutInfo.Version.Minor,
			rData.reviewLayoutInfo.Version.Major, rData.reviewLayoutInfo.Version.Minor, rData.pathwayID, rData.skuType); err != nil {
			www.APIInternalError(w, r, err)
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

	httputil.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": message,
	})
}

type requestData struct {
	intakeLayoutInfo   *layoutInfo
	reviewLayoutInfo   *layoutInfo
	diagnoseLayoutInfo *layoutInfo
	pathwayID          int64
	skuID              *int64
	skuType            string

	// intake/review versioning specific
	intakeUpgradeType common.VersionComponent
	reviewUpgradeType common.VersionComponent
	patientAppVersion *common.Version
	doctorAppVersion  *common.Version
	platform          common.Platform

	// parsed layouts
	intakeLayout   *info_intake.InfoIntakeLayout
	reviewLayout   *info_intake.DVisitReviewSectionListView
	reviewJS       map[string]interface{}
	diagnoseLayout *info_intake.DiagnosisIntake
}

func (rData *requestData) populateTemplatesAndPathway(r *http.Request, dataAPI api.DataAPI) error {
	var pathwayTag string
	var skuStr string
	var numTemplates int64
	var err error

	layouts := map[string]*layoutInfo{
		intake:   nil,
		review:   nil,
		diagnose: nil,
	}

	// Read the uploaded layouts and get pathway tag
	for name := range layouts {
		if file, fileHeader, err := r.FormFile(name); err != http.ErrMissingFile {
			if err != nil {
				return err
			}

			data, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}

			// Parse the json to get the pathway and version which is needed to fetch
			// active templates.
			var js map[string]interface{}
			if err = json.Unmarshal(data, &js); err != nil {
				return err
			}

			if v, ok := js["version"]; ok {
				versionInfo := strings.Split(v.(string), `.`)
				if len(versionInfo) != 3 {
					return fmt.Errorf("Unknown version info attached to blob %v", v)
				}
				layouts[name] = &layoutInfo{
					Data:     data,
					FileName: fmt.Sprintf("%s-%s-%s-%s.json", name, versionInfo[0], versionInfo[1], versionInfo[2]),
				}
			} else {
				layouts[name] = &layoutInfo{
					Data:     data,
					FileName: fileHeader.Filename,
				}
			}

			var pathway string
			if v, ok := js["health_condition"]; ok {
				switch x := v.(type) {
				case string: // patient intake and doctor review
					pathway = x
				case map[string]interface{}: // diagnosis has it at the second level
					if c, ok := x["health_condition"].(string); ok {
						pathway = c
					}
				}
			}

			if pathway == "" {
				return errors.New("pathway is not set")
			}

			if pathwayTag == "" {
				pathwayTag = pathway
			} else if pathwayTag != pathway {
				return errors.New("Health conditions for all layouts must match")
			}

			// Get the sku from the layout
			var s string
			if v, ok := js["cost_item_type"]; ok {
				s = v.(string)
			}
			if skuStr == "" {
				skuStr = s
			} else if s != "" && skuStr != s {
				return errors.New("cost item types do not match across patient and doctor layouts")
			}

			numTemplates++
		}
	}

	// sku is required to be specified when dealing with an intake or review layout
	if layouts[intake] != nil || layouts[review] != nil {
		sku, err := dataAPI.SKU(skuStr)
		if err != nil {
			return err
		}
		rData.skuType = sku.Type
		rData.skuID = &sku.ID
	}

	pathway, err := dataAPI.PathwayForTag(pathwayTag, api.PONone)
	if err != nil {
		return err
	}
	rData.pathwayID = pathway.ID

	if numTemplates == 0 {
		return errors.New("No layouts attached")
	}

	// iterate through the layouts once more to determine the patch type and the incoming version
	// now that we have the condition and the sku type
	for name, layout := range layouts {
		if layout == nil {
			continue
		}

		layout.UpgradeType, layout.Version, err = determinePatchType(layout.FileName, name, rData.pathwayID, rData.skuID, dataAPI)
		if err != nil {
			return err
		}
	}

	// identify the specific layoutInfos to make it easier to do layout specific validation
	rData.intakeLayoutInfo, rData.reviewLayoutInfo, rData.diagnoseLayoutInfo =
		layouts[intake], layouts[review], layouts[diagnose]

	return nil
}

func (rData *requestData) validateUpgradePathsAndLayouts(r *http.Request, dataAPI api.DataAPI) error {

	// nothing to do since there are no upgrades for the intake/review
	if rData.intakeLayoutInfo == nil && rData.reviewLayoutInfo == nil {
		return nil
	}

	if rData.intakeLayoutInfo != nil {
		rData.intakeUpgradeType = rData.intakeLayoutInfo.UpgradeType
	}
	if rData.reviewLayoutInfo != nil {
		rData.reviewUpgradeType = rData.reviewLayoutInfo.UpgradeType
	}

	// ensure that we have the right combination of upgrades
	switch rData.intakeUpgradeType {
	case common.Major, common.Minor:
		if !(rData.reviewUpgradeType == common.Major || rData.reviewUpgradeType == common.Minor) {
			return errors.New("A major/minor upgrade for intake requires a major/minor upgrade on the review")
		}
	default:
		if rData.reviewUpgradeType == common.Major || rData.reviewUpgradeType == common.Minor {
			return errors.New("A major/minor upgrade for review requires a major/minor upgrade on the intake")
		}
	}

	// ensure that app version information is specified and valid
	// if we are dealing with MAJOR upgrades
	var err error
	if rData.intakeUpgradeType == common.Major {
		patientAppVersion := r.FormValue("patient_app_version")
		if patientAppVersion == "" {
			return errors.New("patient_app_version must be specified for MAJOR upgrades")
		}

		rData.patientAppVersion, err = common.ParseVersion(patientAppVersion)
		if err != nil {
			return errors.New(err.Error())
		}

		currentPatientAppVersion, err := dataAPI.LatestAppVersionSupported(rData.pathwayID, rData.skuID, rData.platform, api.RolePatient, api.ReviewPurpose)
		if err != nil && !api.IsErrNotFound(err) {
			return err
		} else if rData.patientAppVersion.LessThan(currentPatientAppVersion) {
			return fmt.Errorf("the patient app version for the major upgrade has to be greater than %s", currentPatientAppVersion.String())
		}

		if err := parsePlatform(r, rData); err != nil {
			return err
		}
	}
	if rData.reviewUpgradeType == common.Major {
		doctorAppVersion := r.FormValue("doctor_app_version")
		if doctorAppVersion == "" {
			return errors.New("doctor_app_version must be specified for MAJOR upgrades")
		}

		rData.doctorAppVersion, err = common.ParseVersion(doctorAppVersion)
		if err != nil {
			return err
		}

		currentDoctorAppVersion, err := dataAPI.LatestAppVersionSupported(rData.pathwayID, rData.skuID, rData.platform, api.RoleDoctor, api.ConditionIntakePurpose)
		if err != nil && !api.IsErrNotFound(err) {
			return err
		} else if rData.doctorAppVersion.LessThan(currentDoctorAppVersion) {
			return fmt.Errorf("the doctor app version for the major upgrade has to be greater than %s", currentDoctorAppVersion.String())
		}

		if err := parsePlatform(r, rData); err != nil {
			return err
		}
	}

	// Parse the layouts and get active layout for anything not uploaded
	var patchUpgrade bool

	// Patient Intake
	if rData.intakeLayoutInfo != nil {
		if err = json.Unmarshal(rData.intakeLayoutInfo.Data, &rData.intakeLayout); err != nil {
			return err
		}

		// validate the intakeLayout against the existing reviewLayout,
		// given that we are dealing with a patch version upgrade for the intake layout
		if rData.intakeUpgradeType == common.Patch {
			patchUpgrade = true
			var rJS map[string]interface{}
			var reviewLayout *info_intake.DVisitReviewSectionListView
			data, _, err := dataAPI.ReviewLayoutForIntakeLayoutVersion(rData.intakeLayoutInfo.Version.Major,
				rData.intakeLayoutInfo.Version.Minor, rData.pathwayID, rData.skuType)
			if err != nil {
				return err
			} else if err := json.Unmarshal(data, &rJS); err != nil {
				return err
			} else if err := decodeReviewJSIntoLayout(rJS, &reviewLayout); err != nil {
				return err
			} else if err := validateIntakeReviewPair(r, rData.intakeLayout, rJS, reviewLayout, dataAPI); err != nil {
				return err
			}
		}
	}

	// Doctor review
	if rData.reviewLayoutInfo != nil {
		if err := json.Unmarshal(rData.reviewLayoutInfo.Data, &rData.reviewJS); err != nil {
			return err
		}

		if err := decodeReviewJSIntoLayout(rData.reviewJS, &rData.reviewLayout); err != nil {
			return err
		}

		// validate the reviewLayout against the existing intakeLayout that it maps to,
		// given that we are dealing with a patch version upgrade for the review layout
		if rData.reviewUpgradeType == common.Patch {
			patchUpgrade = true
			var infoIntake *info_intake.InfoIntakeLayout
			data, _, err := dataAPI.IntakeLayoutForReviewLayoutVersion(rData.reviewLayoutInfo.Version.Major,
				rData.reviewLayoutInfo.Version.Minor, rData.pathwayID, rData.skuType)
			if err != nil {
				return err
			} else if err := json.Unmarshal(data, &infoIntake); err != nil {
				return err
			} else if err := validateIntakeReviewPair(r, infoIntake, rData.reviewJS, rData.reviewLayout, dataAPI); err != nil {
				return err
			}
		}
	}

	if !patchUpgrade {
		// only validate the intake/review pair provided in the request parameters if dealing with a non-patch upgrade
		// Validate the intake/review layouts
		return validateIntakeReviewPair(r, rData.intakeLayout, rData.reviewJS, rData.reviewLayout, dataAPI)
	}

	return nil
}

func (rData *requestData) parseAndValidateDiagnosisLayout(r *http.Request, dataAPI api.DataAPI) error {
	if rData.diagnoseLayoutInfo == nil {
		return nil
	}

	if err := json.Unmarshal(rData.diagnoseLayoutInfo.Data, &rData.diagnoseLayout); err != nil {
		return err
	}

	if err := api.FillDiagnosisIntake(rData.diagnoseLayout, dataAPI, api.LanguageIDEnglish); err != nil {
		// TODO: this could be a validation error (unknown question or answer) or an internal error.
		// There's currently no easy way to tell the difference. This is ok for now since this is
		// an admin endpoint.
		return err
	}
	return nil
}

func decodeReviewJSIntoLayout(reviewJS map[string]interface{}, reviewLayout **info_intake.DVisitReviewSectionListView) error {
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:   reviewLayout,
		TagName:  "json",
		Registry: *info_intake.DVisitReviewViewTypeRegistry,
	})
	if err != nil {
		return err
	}

	if err := d.Decode(reviewJS["visit_review"]); err != nil {
		return err
	}

	return nil
}

func validateIntakeReviewPair(r *http.Request, intakeLayout *info_intake.InfoIntakeLayout, reviewJS map[string]interface{},
	reviewLayout *info_intake.DVisitReviewSectionListView, dataAPI api.DataAPI) error {

	if err := api.FillIntakeLayout(intakeLayout, dataAPI, api.LanguageIDEnglish); err != nil {
		// TODO: this could be a validation error (unknown question or answer) or an internal error.
		// There's currently no easy way to tell the difference. This is ok for now since this is
		// an admin endpoint.
		return err
	}
	if err := validatePatientLayout(intakeLayout); err != nil {
		return err
	}
	if err := compareQuestions(intakeLayout, reviewJS); err != nil {
		return err

	}

	// Make sure the review layout renders
	context, err := reviewContext(intakeLayout)
	if err != nil {
		return err
	}
	if _, err = reviewLayout.Render(common.NewViewContext(context)); err != nil {
		return err
	}

	return nil
}

// validateVersionedFileName validates an incoming layout file to be of the format
// layoutType-X-Y-Z.json
func validateVersionedFileName(fileName, layoutType string) (*common.Version, error) {
	invalidFileFormat := fmt.Errorf("Unknown versioned filename. Should be of the form condition-X-Y-Z.json or review-X-Y-Z.json.")
	endIndex := strings.Index(fileName, ".json")
	if endIndex < 0 {
		return nil, invalidFileFormat
	}

	i := strings.Index(fileName, layoutType)
	if i < 0 {
		return nil, invalidFileFormat
	}

	version, err := common.ParseVersion(fileName[i+len(layoutType)+1 : endIndex])
	if err != nil {
		return nil, invalidFileFormat
	}

	return version, nil
}

// determinePatchType identifies the type of versioning the layout is to undergo
// based on the expected version to upgrade to in the name of the file
func determinePatchType(fileName, layoutType string, pathwayID int64, skuID *int64, dataAPI api.DataAPI) (common.VersionComponent, *common.Version, error) {
	var role, purpose string
	switch layoutType {
	case review:
		role, purpose = api.RoleDoctor, api.ReviewPurpose
	case intake:
		role, purpose = api.RolePatient, api.ConditionIntakePurpose
	case diagnose:
		role, purpose = api.RoleDoctor, api.DiagnosePurpose
	default:
		return common.InvalidVersionComponent, nil, fmt.Errorf("Unknown layoutType: %s", layoutType)
	}

	incomingVersion, err := validateVersionedFileName(fileName, layoutType)
	if err != nil {
		return common.InvalidVersionComponent, nil, nil
	}

	determineLatestVersion := func(versionInfo *api.VersionInfo) error {
		layoutVersion, err := dataAPI.LayoutTemplateVersionBeyondVersion(versionInfo, role, purpose, pathwayID, skuID)
		if err != nil {
			return err
		}
		if !layoutVersion.Version.LessThan(incomingVersion) {
			return fmt.Errorf("Incoming verison is older than existing version in the database for role %s and purpose %s", role, purpose)
		}
		return nil
	}

	// determine the latest layout version for the (MAJOR,MINOR) combination
	if err := determineLatestVersion(
		&api.VersionInfo{
			Major: &(incomingVersion.Major),
			Minor: &(incomingVersion.Minor),
		},
	); err == nil {
		return common.Patch, incomingVersion, nil
	} else if !api.IsErrNotFound(err) {
		return common.InvalidVersionComponent, nil, err
	}

	// determine the latest layout version for the MAJOR version component
	if err := determineLatestVersion(
		&api.VersionInfo{
			Major: &(incomingVersion.Major),
		},
	); err == nil {
		return common.Minor, incomingVersion, nil
	} else if !api.IsErrNotFound(err) {
		return common.InvalidVersionComponent, nil, err
	}

	// determine the latest layout version in the database
	if err = determineLatestVersion(nil); err != nil && !api.IsErrNotFound(err) {
		return common.InvalidVersionComponent, nil, err
	}

	return common.Major, incomingVersion, nil
}

func parsePlatform(r *http.Request, rData *requestData) error {
	platform := r.FormValue("platform")
	if platform == "" {
		return errors.New("platform must be specified for MAJOR upgrades")
	}

	var err error
	if rData.platform, err = common.GetPlatform(platform); err != nil {
		return err
	}

	return nil
}

type errorList struct {
	Errors []string
}

func (e *errorList) Error() string {
	return "layout.validate: " + strings.Join([]string(e.Errors), ", ")
}

func (e *errorList) Len() int {
	return len(e.Errors)
}

func (e *errorList) Append(err string) {
	e.Errors = append(e.Errors, err)
}

func validateQuestion(que *info_intake.Question, path string, errors *errorList) {
	if que.QuestionTag == "" {
		errors.Append(fmt.Sprintf("%s missing 'question'", path))
	}
	switch que.QuestionType {
	case info_intake.QuestionTypeMultipleChoice,
		info_intake.QuestionTypeSingleSelect,
		info_intake.QuestionTypeSegmentedControl:
		if len(que.PotentialAnswers) == 0 {
			errors.Append(fmt.Sprintf("%s missing potential answers", path))
		}
	case info_intake.QuestionTypePhotoSection:
		if len(que.PhotoSlots) == 0 {
			errors.Append(fmt.Sprintf("%s missing photo slots", path))
		}
	case info_intake.QuestionTypeFreeText,
		info_intake.QuestionTypeAutocomplete:
		if len(que.PotentialAnswers) != 0 {
			errors.Append(fmt.Sprintf("%s should not have potential answers", path))
		}
	}
	if c := que.SubQuestionsConfig; c != nil {
		for i, q := range c.Questions {
			validateQuestion(q, fmt.Sprintf("%s.subquestion[%d]", path, i), errors)
		}
	}
	if que.ConditionBlock != nil {
		validateCondition(que.ConditionBlock, fmt.Sprintf("%s.condition", path), errors)
	}
}

func validateCondition(cond *info_intake.Condition, path string, errors *errorList) {
	switch cond.OperationTag {
	case "":
		errors.Append(fmt.Sprintf("%s missing op in condition", path))
	case "answer_contains_any", "answer_contains_all", "answer_equals_exact", "answer_equals":
		if cond.QuestionTag == "" {
			errors.Append(fmt.Sprintf("%s missing question for '%s' condition", path, cond.OperationTag))
		}
		if len(cond.PotentialAnswersTags) == 0 {
			errors.Append(fmt.Sprintf("%s missing potential answers for '%s' condition", path, cond.OperationTag))
		}
	case "gender_equals":
		if cond.GenderField == "" {
			errors.Append(fmt.Sprintf("%s missing gender for '%s' condition", path, cond.OperationTag))
		}
	case "and", "or", "not":
		for _, cond := range cond.Operands {
			validateCondition(cond, fmt.Sprintf("%s.%s", path, cond.OperationTag), errors)
		}
	default:
		errors.Append(fmt.Sprintf("%s unknown condition op '%s'", path, cond.OperationTag))
	}
}

func validatePatientLayout(layout *info_intake.InfoIntakeLayout) error {
	errors := &errorList{}
	if len(layout.Sections) == 0 {
		errors.Append("layout contains no sections")
	}
	if layout.PathwayTag == "" {
		errors.Append("pathway tag not set")
	}
	for secIdx, sec := range layout.Sections {
		path := fmt.Sprintf("section[%d]", secIdx)
		if sec.SectionTag == "" {
			errors.Append(fmt.Sprintf("%s missing 'section'", path))
		}
		if sec.SectionID == "" {
			errors.Append(fmt.Sprintf("%s missing 'section_id'", path))
		}
		if sec.SectionTitle == "" {
			errors.Append(fmt.Sprintf("%s missing 'section_title'", path))
		}
		if len(sec.Screens) == 0 {
			errors.Append(fmt.Sprintf("%s has no screens", path))
		}
		for scrIdx, scr := range sec.Screens {
			switch scr.ScreenType {
			case "screen_type_pharmacy", "screen_type_triage", "screen_type_warning_popup", "screen_type_generic_popup":
				continue
			}

			pth := fmt.Sprintf("%s.screen[%d]", path, scrIdx)
			if scr.ConditionBlock != nil {
				validateCondition(scr.ConditionBlock, fmt.Sprintf("%s.condition", pth), errors)
			}
			if len(scr.Questions) == 0 {
				errors.Append(fmt.Sprintf("%s has no questions", path))
			}
			for queIdx, que := range scr.Questions {
				validateQuestion(que, fmt.Sprintf("%s.question[%d:%s]", pth, queIdx, que.QuestionTag), errors)
			}
		}
	}
	if errors.Len() != 0 {
		return errors
	}
	return nil
}

// Find all values that are strings that start with "q_" which represent a question
func questionMap(in interface{}, out map[string]bool) {
	switch v := in.(type) {
	case string:
		if strings.HasPrefix(v, "q_") {
			if idx := strings.IndexByte(v, ':'); idx > 0 {
				out[v[:idx]] = true
			}
		}
	case []interface{}:
		for _, v2 := range v {
			questionMap(v2, out)
		}
	case map[string]interface{}:
		for _, v2 := range v {
			questionMap(v2, out)
		}
	}
}

func reviewContext(patientLayout *info_intake.InfoIntakeLayout) (map[string]interface{}, error) {
	context := make(map[string]interface{})
	context["patient_visit_alerts"] = []string{"ALERT"}
	context["visit_message"] = "message"
	for _, sec := range patientLayout.Sections {
		if len(sec.Questions) != 0 {
			return nil, fmt.Errorf("Don't support questions in a section outside of a screen")
		}
		for _, scr := range sec.Screens {
			for _, que := range scr.Questions {
				switch que.QuestionType {
				case info_intake.QuestionTypePhotoSection:
					photoList := make([]info_intake.TitlePhotoListData, len(que.PhotoSlots))
					for i, slot := range que.PhotoSlots {
						photoList[i] = info_intake.TitlePhotoListData{
							Title:  slot.Name,
							Photos: []info_intake.PhotoData{},
						}
					}
					context["patient_visit_photos"] = photoList
					context[que.QuestionTag+":photos"] = photoList
				case info_intake.QuestionTypeSingleSelect,
					info_intake.QuestionTypeSingleEntry,
					info_intake.QuestionTypeFreeText,
					info_intake.QuestionTypeSegmentedControl:

					context[que.QuestionTag+":question_summary"] = "Summary"
					context[que.QuestionTag+":answers"] = "Answer"
				case info_intake.QuestionTypeMultipleChoice:
					if sub := que.SubQuestionsConfig; sub != nil {
						data := []info_intake.TitleSubItemsDescriptionContentData{
							info_intake.TitleSubItemsDescriptionContentData{
								Title: "Title",
								SubItems: []*info_intake.DescriptionContentData{
									&info_intake.DescriptionContentData{
										Description: "Description",
										Content:     "Content",
									},
								},
							},
						}
						context[que.QuestionTag+":question_summary"] = "Summary"
						context[que.QuestionTag+":answers"] = data
					} else {
						context[que.QuestionTag+":question_summary"] = "Summary"
						context[que.QuestionTag+":answers"] = []info_intake.CheckedUncheckedData{
							{Value: "Value", IsChecked: true},
						}
					}
				case info_intake.QuestionTypeAutocomplete:
					data := []info_intake.TitleSubItemsDescriptionContentData{
						info_intake.TitleSubItemsDescriptionContentData{
							Title: "Title",
							SubItems: []*info_intake.DescriptionContentData{
								&info_intake.DescriptionContentData{
									Description: "Description",
									Content:     "Content",
								},
							},
						},
					}
					context[que.QuestionTag+":question_summary"] = "Summary"
					context[que.QuestionTag+":answers"] = data
				default:
					return nil, fmt.Errorf("Unknown question type '%s'", que.QuestionType)
				}
			}
		}
	}
	return context, nil
}

func compareQuestions(intakeLayout *info_intake.InfoIntakeLayout, reviewJS map[string]interface{}) error {
	intakeQuestions := map[string]bool{}
	conditionQuestions := map[string]bool{}
	for _, sec := range intakeLayout.Sections {
		if len(sec.Questions) != 0 {
			return fmt.Errorf("Questions in a section outside of a screen unsupported")
		}
		for _, scr := range sec.Screens {
			for _, que := range scr.Questions {
				intakeQuestions[que.QuestionTag] = true
				if con := que.ConditionBlock; con != nil {
					conditionQuestions[con.QuestionTag] = true
				}
			}
		}
	}

	reviewQuestions := map[string]bool{}
	questionMap(reviewJS, reviewQuestions)

	for q := range intakeQuestions {
		if !reviewQuestions[q] {
			// It's ok if the question doesn't show up in the review layout
			// if it's used in a condition.
			if !conditionQuestions[q] {
				return fmt.Errorf("Question '%s' in intake but not in review layout", q)
			}
		}
		delete(reviewQuestions, q)
	}
	if len(reviewQuestions) != 0 {
		s := make([]string, 0, len(reviewQuestions))
		for q := range reviewQuestions {
			s = append(s, q)
		}
		return fmt.Errorf("Question(s) '%s' in review layout but not in intake", strings.Join(s, ","))
	}

	return nil
}
