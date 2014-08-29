package layout

import "testing"

func TestValidVersionedFileName(t *testing.T) {
	testValidVersionedFileName("review-1-0-0.json", review, t)
	testValidVersionedFileName("intake-1-0-0.json", intake, t)
	testValidVersionedFileName("intake-100-100-999.json", intake, t)
	testValidVersionedFileName("review-0-0-1.json", review, t)
}

func TestInvalidVersionedFileName(t *testing.T) {
	testInvalidVersionedFileName("r", review, t)
	testInvalidVersionedFileName("review-1-0-0.jso", review, t)
	testInvalidVersionedFileName("intake-", intake, t)
	testInvalidVersionedFileName("bintake-100-100-999.json", intake, t)
	testInvalidVersionedFileName("review-0-0-1.json", intake, t)
}

func testValidVersionedFileName(fileName, layoutType string, t *testing.T) {
	if _, err := validateVersionedFileName(fileName, layoutType); err != nil {
		t.Fatal(err)
	}
}

func testInvalidVersionedFileName(fileName, layoutType string, t *testing.T) {
	if _, err := validateVersionedFileName(fileName, layoutType); err == nil {
		t.Fatal("Expected error but got none")
	}
}
