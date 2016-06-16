package media

import (
	"regexp"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
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

func TestParseMediaID(t *testing.T) {
	mediaID, err := ParseMediaID("s3://us-east-1/test-baymax-storage/media/12345")
	test.OK(t, err)
	test.Equals(t, "12345", mediaID)

	mediaID, err = ParseMediaID("12345")
	test.OK(t, err)
	test.Equals(t, "12345", mediaID)
}
