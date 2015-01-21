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
		return apiservice.NewValidationError("platform must be specified for MAJOR upgrades")
	}

	var err error
	if rData.platform, err = common.GetPlatform(platform); err != nil {
		return apiservice.NewValidationError(err.Error())
	}

	return nil
}
