package sync

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestIDForSource(t *testing.T) {
	id, err := IDForSource("csv_12345")
	test.OK(t, err)
	test.Equals(t, "12345", id)

	id, err = IDForSource("drchrono_12345")
	test.OK(t, err)
	test.Equals(t, "12345", id)

	id, err = IDForSource("hint_12345")
	test.OK(t, err)
	test.Equals(t, "12345", id)

	id, err = IDForSource("elation_12345")
	test.OK(t, err)
	test.Equals(t, "12345", id)
}

func TestExternalIDFromSource(t *testing.T) {
	test.Equals(t, ExternalIDFromSource("12345", SOURCE_CSV), "csv_12345")
	test.Equals(t, ExternalIDFromSource("12345", SOURCE_HINT), "hint_12345")
	test.Equals(t, ExternalIDFromSource("12345", SOURCE_ELATION), "elation_12345")
	test.Equals(t, ExternalIDFromSource("12345", SOURCE_DRCHRONO), "drchrono_12345")
}
