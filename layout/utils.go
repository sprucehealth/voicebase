package layout

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

const (
	maxMemory = 2 * 1024 * 1024 // MB
	intake    = "intake"
	review    = "review"
	diagnose  = "diagnose"
)

func validateVersionedFileName(fileName, layoutType string) (*common.Version, error) {
	invalidFileFormat := fmt.Errorf("Unknown versioned filename. Should be of the form condition-X-Y-Z.json or review-X-Y-Z.json.")
	endIndex := strings.Index(fileName, ".json")
	if endIndex < 0 {
		return nil, invalidFileFormat
	}

	if i := strings.Index(fileName, layoutType); i < 0 {
		return nil, invalidFileFormat
	}

	version, err := common.ParseVersion(fileName[len(layoutType)+1 : endIndex])
	if err != nil {
		return nil, invalidFileFormat
	}

	return version, nil
}

func determinePatchType(fileName, layoutType string, dataAPI api.DataAPI) (common.VersionComponent, *common.Version, error) {

	var role, purpose string
	switch layoutType {
	case review:
		role, purpose = api.DOCTOR_ROLE, api.ReviewPurpose
	case intake:
		role, purpose = api.PATIENT_ROLE, api.ConditionIntakePurpose
	case diagnose:
		role, purpose = api.DOCTOR_ROLE, api.DiagnosePurpose
	default:
		return common.InvalidVersionComponent, nil, fmt.Errorf("Unknown layoutType: %s", layoutType)
	}

	incomingVersion, err := validateVersionedFileName(fileName, layoutType)
	if err != nil {
		return common.InvalidVersionComponent, nil, nil
	}

	determineLatestVersion := func(versionInfo *api.VersionInfo) error {
		layoutVersion, err := dataAPI.LayoutTemplateVersionBeyondVersion(versionInfo, role, purpose)
		if err != api.NoRowsError && err != nil {
			return err
		} else if err == nil {
			if !layoutVersion.Version.LessThan(incomingVersion) {
				return fmt.Errorf("Incoming verison is older than existing version in the database for role %s and purpose %s", role, purpose)
			}
		}
		return err
	}

	// determine the latest layout version for the (MAJOR,MINOR) combination
	if err := determineLatestVersion(
		&api.VersionInfo{
			Major: &(incomingVersion.Major),
			Minor: &(incomingVersion.Minor),
		}); err != nil && err != api.NoRowsError {
		return common.InvalidVersionComponent, nil, err
	} else if err == nil {
		return common.Patch, incomingVersion, nil
	}

	// determine the latest layout version for the MAJOR version component
	if err := determineLatestVersion(
		&api.VersionInfo{
			Major: &(incomingVersion.Major),
		}); err != nil && err != api.NoRowsError {
		return common.InvalidVersionComponent, nil, err
	} else if err == nil {
		return common.Minor, incomingVersion, nil
	}

	// determine the latest layout version in the database
	if err := determineLatestVersion(nil); err != nil && err != api.NoRowsError {
		return common.InvalidVersionComponent, nil, err
	}

	return common.Major, incomingVersion, nil
}

func parsePlatform(r *http.Request, rData *requestData) error {
	platform := r.FormValue("platform")
	if platform == "" {
		return apiservice.NewValidationError("platform must be specified for MAJOR upgrades", r)
	}

	var err error
	if rData.platform, err = common.GetPlatform(platform); err != nil {
		return apiservice.NewValidationError(err.Error(), r)
	}

	return nil
}
