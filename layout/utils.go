package layout

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common"
)

const (
	maxMemory = 2 * 1024 * 1024 // MB
	intake    = "intake"
	review    = "review"
	diagnose  = "diagnose"
)

func validateVersionedFileName(fileName, layoutType string) error {
	invalidFileFormat := fmt.Errorf("Unknown versioned filename. Should be of the form condition-X-Y-Z.json or review-X-Y-Z.json.")
	endIndex := strings.Index(fileName, ".json")
	if endIndex < 0 {
		return invalidFileFormat
	}

	if i := strings.Index(fileName, layoutType); i < 0 {
		return invalidFileFormat
	}

	_, err := common.ParseVersion(strings.Replace(fileName[len(layoutType)+1:endIndex], "-", ".", -1))
	if err != nil {
		return invalidFileFormat
	}

	return nil
}
