package layout

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/sku"
)

type requestData struct {
	intakeLayoutInfo   *layoutInfo
	reviewLayoutInfo   *layoutInfo
	diagnoseLayoutInfo *layoutInfo
	pathwayID          int64
	skuID              *int64
	skuType            sku.SKU

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
				return apiservice.NewValidationError(err.Error())
			}

			data, err := ioutil.ReadAll(file)
			if err != nil {
				return apiservice.NewValidationError(err.Error())
			}

			layouts[name] = &layoutInfo{
				Data:     data,
				FileName: fileHeader.Filename,
			}

			// Parse the json to get the pathway which is needed to fetch
			// active templates.

			var js map[string]interface{}
			if err = json.Unmarshal(data, &js); err != nil {
				return apiservice.NewValidationError(err.Error())
			}
			var pathawy string
			if v, ok := js["health_condition"]; ok {
				switch x := v.(type) {
				case string: // patient intake and doctor review
					pathawy = x
				case map[string]interface{}: // diagnosis has it at the second level
					if c, ok := x["health_condition"].(string); ok {
						pathawy = c
					}
				}
			}
			if pathawy == "" {
				return apiservice.NewValidationError("pathway is not set")
			}

			if pathwayTag == "" {
				pathwayTag = pathawy
			} else if pathwayTag != pathawy {
				return apiservice.NewValidationError("Health conditions for all layouts must match")
			}

			// Get the sku from the layout
			var s string
			if v, ok := js["cost_item_type"]; ok {
				s = v.(string)
			}
			if skuStr == "" {
				skuStr = s
			} else if s != "" && skuStr != s {
				return apiservice.NewValidationError("cost item types do not match across patient and doctor layouts")
			}

			numTemplates++
		}
	}

	// sku is required to be specified when dealing with an intake or review layout
	if layouts[intake] != nil || layouts[review] != nil {
		rData.skuType, err = sku.GetSKU(skuStr)
		if err != nil {
			return err
		}
		sID, err := dataAPI.SKUID(rData.skuType)
		if err != nil {
			return err
		}
		rData.skuID = &sID
	}

	pathway, err := dataAPI.PathwayForTag(pathwayTag)
	if err != nil {
		return err
	}
	rData.pathwayID = pathway.ID

	if numTemplates == 0 {
		return apiservice.NewValidationError("No layouts attached")
	}

	// iterate through the layouts once more to determine the patch type and the incoming version
	// now that we have the condition and the sku type
	for name, layout := range layouts {
		if layout == nil {
			continue
		}

		layout.UpgradeType, layout.Version, err = determinePatchType(layout.FileName, name, rData.pathwayID, rData.skuID, dataAPI)
		if err != nil {
			return apiservice.NewValidationError(err.Error())
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
			return apiservice.NewValidationError("A major/minor upgrade for intake requires a major/minor upgrade on the review")
		}
	default:
		if rData.reviewUpgradeType == common.Major || rData.reviewUpgradeType == common.Minor {
			return apiservice.NewValidationError("A major/minor upgrade for review requires a major/minor upgrade on the intake")
		}
	}

	// ensure that app version information is specified and valid
	// if we are dealing with MAJOR upgrades
	var err error
	if rData.intakeUpgradeType == common.Major {
		patientAppVersion := r.FormValue("patient_app_version")
		if patientAppVersion == "" {
			return apiservice.NewValidationError("patient_app_version must be specified for MAJOR upgrades")
		}

		rData.patientAppVersion, err = common.ParseVersion(patientAppVersion)
		if err != nil {
			return apiservice.NewValidationError(err.Error())
		}

		currentPatientAppVersion, err := dataAPI.LatestAppVersionSupported(rData.pathwayID, rData.skuID, rData.platform, api.PATIENT_ROLE, api.ReviewPurpose)
		if err != nil && !api.IsErrNotFound(err) {
			return err
		} else if rData.patientAppVersion.LessThan(currentPatientAppVersion) {
			return apiservice.NewValidationError(fmt.Sprintf("the patient app version for the major upgrade has to be greater than %s", currentPatientAppVersion.String()))
		}

		if err := parsePlatform(r, rData); err != nil {
			return err
		}
	}
	if rData.reviewUpgradeType == common.Major {
		doctorAppVersion := r.FormValue("doctor_app_version")
		if doctorAppVersion == "" {
			return apiservice.NewValidationError("doctor_app_version must be specified for MAJOR upgrades")
		}

		rData.doctorAppVersion, err = common.ParseVersion(doctorAppVersion)
		if err != nil {
			return apiservice.NewValidationError(err.Error())
		}

		currentDoctorAppVersion, err := dataAPI.LatestAppVersionSupported(rData.pathwayID, rData.skuID, rData.platform, api.DOCTOR_ROLE, api.ConditionIntakePurpose)
		if err != nil && !api.IsErrNotFound(err) {
			return err
		} else if rData.doctorAppVersion.LessThan(currentDoctorAppVersion) {
			return apiservice.NewValidationError(fmt.Sprintf("the doctor app version for the major upgrade has to be greater than %s", currentDoctorAppVersion.String()))
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
			return apiservice.NewValidationError("Failed to parse json: " + err.Error())
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
			return apiservice.NewValidationError("Failed to parse json: " + err.Error())
		}

		if decodeReviewJSIntoLayout(rData.reviewJS, &rData.reviewLayout); err != nil {
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
		return apiservice.NewValidationError("Failed to parse json: " + err.Error())
	}

	if err := api.FillDiagnosisIntake(rData.diagnoseLayout, dataAPI, api.EN_LANGUAGE_ID); err != nil {
		// TODO: this could be a validation error (unknown question or answer) or an internal error.
		// There's currently no easy way to tell the difference. This is ok for now since this is
		// an admin endpoint.
		return apiservice.NewValidationError(err.Error())
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

	if err := api.FillIntakeLayout(intakeLayout, dataAPI, api.EN_LANGUAGE_ID); err != nil {
		// TODO: this could be a validation error (unknown question or answer) or an internal error.
		// There's currently no easy way to tell the difference. This is ok for now since this is
		// an admin endpoint.
		return apiservice.NewValidationError(err.Error())
	}
	if err := validatePatientLayout(intakeLayout); err != nil {
		return apiservice.NewValidationError(err.Error())
	}
	if err := compareQuestions(intakeLayout, reviewJS); err != nil {
		return apiservice.NewValidationError(err.Error())

	}

	// Make sure the review layout renders
	context, err := reviewContext(intakeLayout)
	if err != nil {
		return apiservice.NewValidationError(err.Error())
	}
	if _, err = reviewLayout.Render(common.NewViewContext(context)); err != nil {
		return apiservice.NewValidationError(err.Error())
	}

	return nil
}
