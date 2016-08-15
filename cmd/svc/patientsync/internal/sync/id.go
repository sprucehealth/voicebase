package sync

import (
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
)

// ExternalIDFromSource creates an ID that contains the source information
// in the ID for easy referencing and scoping.
func ExternalIDFromSource(id string, source Source) string {
	var prefix string
	switch source {
	case SOURCE_CSV:
		prefix = "csv"
	case SOURCE_DRCHRONO:
		prefix = "drchrono"
	case SOURCE_ELATION:
		prefix = "elation"
	case SOURCE_HINT:
		prefix = "hint"
	default:
		prefix = "unknown"
	}

	return prefix + "_" + id
}

// IDForSource returns the actual ID for the source from the
// externalID populated for internal system referencing.
func IDForSource(externalID string) (string, error) {
	prefix := strings.IndexRune(externalID, '_')
	if prefix == -1 {
		return "", errors.Errorf("malformed id %s", externalID)
	}
	return externalID[prefix+1:], nil
}
