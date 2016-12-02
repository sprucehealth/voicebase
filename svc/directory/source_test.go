package directory

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/test"
)

func TestFlattenEntitySource(t *testing.T) {
	s := &EntitySource{Type: EntitySource_PRACTICE_CODE, Data: "123456"}
	test.Equals(t, "PRACTICE_CODE:123456", FlattenEntitySource(s))
}

func TestParseEntitySource(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s := &EntitySource{Type: EntitySource_PRACTICE_CODE, Data: "123456"}
		ps, err := ParseEntitySource(FlattenEntitySource(s))
		test.OK(t, err)
		test.Equals(t, ps, s)
	})
	t.Run("InvalidType", func(t *testing.T) {
		s := &EntitySource{Type: EntitySource_SourceType(-1), Data: ""}
		_, err := ParseEntitySource(FlattenEntitySource(s))
		test.Equals(t, errors.Cause(err), fmt.Errorf("Invalid EntitySource Type for %s", FlattenEntitySource(s)))
	})
}
