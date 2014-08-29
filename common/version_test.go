package common

import "testing"

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
	if _, err := ParseVersion("a"); err == nil {
		t.Fatal("Expected invalid version")
	}
	if _, err := ParseVersion(".."); err == nil {
		t.Fatal("Expected invalid version")
	}
	if _, err := ParseVersion("0-"); err == nil {
		t.Fatal("Expected invalid version")
	}
	if _, err := ParseVersion("0.5.a"); err == nil {
		t.Fatal("Expected invalid version")
	}
	if _, err := ParseVersion("0.5."); err == nil {
		t.Fatal("Expected invalid version")
	}

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
