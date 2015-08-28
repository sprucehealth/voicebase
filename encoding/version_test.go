package encoding

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestVersionParsing(t *testing.T) {
	// successful parsing
	testVersionParsing("1.2.3", 1, 2, 3, t)
	testVersionParsing("0.2.3", 0, 2, 3, t)
	testVersionParsing("0.2", 0, 2, 0, t)
	testVersionParsing("12", 12, 0, 0, t)
	testVersionParsing("1.0", 1, 0, 0, t)
	testVersionParsing("001.0002.0003", 1, 2, 3, t)
	testVersionParsing("001-0002-0003", 1, 2, 3, t)
	testVersionParsing("0.99", 0, 99, 0, t)
	testVersionParsing("0.9.9", 0, 9, 9, t)
	testVersionParsing("0-9-9", 0, 9, 9, t)

	// unsuccessful parsing
	for _, s := range []string{"a", "..", "0-", "0.5.a", "0.5."} {
		if _, err := ParseVersion(s); err == nil {
			t.Fatalf("Expected invalid version while parsing '%s'", s)
		}
	}
}

func TestVersionMarshalling(t *testing.T) {
	v := Version{
		Major: 10,
		Minor: 2,
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"version": v,
	})
	test.OK(t, err)
	test.Equals(t, `{"version":"10.2.0"}`, string(jsonData))

	text, err := v.MarshalText()
	test.OK(t, err)
	test.Equals(t, "10.2.0", string(text))
}

func TestVersionUnmarshalling(t *testing.T) {
	jsonVersion := `{"version":"0.5.2"}`
	var response struct {
		V *Version `json:"version"`
	}

	err := json.Unmarshal([]byte(jsonVersion), &response)
	test.OK(t, err)
	test.Equals(t, 0, response.V.Major)
	test.Equals(t, 5, response.V.Minor)
	test.Equals(t, 2, response.V.Patch)

	jsonVersion = `{"version":"1.9.8"}`
	var r2 struct {
		V Version `json:"version"`
	}

	err = json.Unmarshal([]byte(jsonVersion), &r2)
	test.OK(t, err)
	test.Equals(t, 1, r2.V.Major)
	test.Equals(t, 9, r2.V.Minor)
	test.Equals(t, 8, r2.V.Patch)

	var v Version
	test.OK(t, v.UnmarshalText([]byte("1.2.3")))
	test.Equals(t, "1.2.3", v.String())
}

func TestVersionLessThan(t *testing.T) {
	testVersionLessThan(
		Version{Major: 1, Minor: 2, Patch: 3},
		Version{Major: 1, Minor: 2, Patch: 4}, t)

	testVersionLessThan(
		Version{Major: 0, Minor: 2, Patch: 3},
		Version{Major: 1, Minor: 2, Patch: 3}, t)

	testVersionLessThan(
		Version{Major: 1, Minor: 0, Patch: 3},
		Version{Major: 1, Minor: 2, Patch: 3}, t)

	testVersionNotLessThan(
		Version{Major: 1, Minor: 2, Patch: 3},
		Version{Major: 1, Minor: 2, Patch: 3}, t)

	testVersionNotLessThan(
		Version{Major: 9, Minor: 2, Patch: 3},
		Version{Major: 1, Minor: 2, Patch: 3}, t)

	testVersionNotLessThan(
		Version{Major: 10, Minor: 0, Patch: 0},
		Version{Major: 1, Minor: 2, Patch: 3}, t)
}

func testVersionLessThan(v1, v2 Version, t *testing.T) {
	if !v1.LessThan(&v2) {
		t.Fatal("Expected v1 to be less than v2")
	}
}

func testVersionNotLessThan(v1, v2 Version, t *testing.T) {
	if v1.LessThan(&v2) {
		t.Fatal("Expected v1 to NOT be less than v2")
	}
}

func testVersionParsing(ver string, major, minor, patch int, t *testing.T) {
	v, err := ParseVersion(ver)
	if err != nil {
		t.Fatal(err)
	} else if v.Major != major {
		t.Fatal("Wrong major component")
	} else if v.Minor != minor {
		t.Fatal("Wrong minor component")
	} else if v.Patch != patch {
		t.Fatal("Wrong patch component")
	}
}

func TestVersionRangeContains(t *testing.T) {
	// Range with no limits should allow everything
	test.Equals(t, true, VersionRange{}.Contains(&Version{1, 0, 0}))
	// Minimum only
	test.Equals(t, true, VersionRange{MinVersion: &Version{1, 0, 0}}.Contains(&Version{1, 0, 0}))
	test.Equals(t, false, VersionRange{MinVersion: &Version{1, 0, 0}}.Contains(&Version{0, 0, 9}))
	// Maximum only
	test.Equals(t, true, VersionRange{MaxVersion: &Version{1, 0, 0}}.Contains(&Version{0, 0, 9}))
	test.Equals(t, false, VersionRange{MaxVersion: &Version{1, 0, 0}}.Contains(&Version{1, 0, 0}))
	// Full range
	test.Equals(t, true, VersionRange{MinVersion: &Version{1, 0, 0}, MaxVersion: &Version{2, 0, 0}}.Contains(&Version{1, 5, 0}))
	test.Equals(t, false, VersionRange{MinVersion: &Version{1, 0, 0}, MaxVersion: &Version{1, 0, 0}}.Contains(&Version{1, 0, 0}))
	test.Equals(t, false, VersionRange{MinVersion: &Version{1, 0, 0}, MaxVersion: &Version{2, 0, 0}}.Contains(&Version{0, 0, 9}))
	test.Equals(t, false, VersionRange{MinVersion: &Version{1, 0, 0}, MaxVersion: &Version{2, 0, 0}}.Contains(&Version{2, 0, 1}))
}
