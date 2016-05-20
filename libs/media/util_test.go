package media

import (
	"regexp"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestIDRegexPattern(t *testing.T) {
	r, err := regexp.Compile(IDRegexPattern)
	test.OK(t, err)
	for i := 0; i < 1000; i++ {
		id, err := NewID()
		test.OK(t, err)
		if !r.MatchString(id) {
			t.Fatalf("ID Regex %s failed to match ID %q", IDRegexPattern, id)
		}
	}
}
