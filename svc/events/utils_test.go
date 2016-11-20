package events

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/directory"
)

func TestBasePkgOfEvent(t *testing.T) {
	e := &directory.EntityUpdatedEvent{}
	test.Equals(t, "directory", basePackageOfEvent(e))
	e1 := directory.EntityUpdatedEvent{}
	test.Equals(t, "directory", basePackageOfEvent(e1))
}
