package apiservice

import "testing"

func TestValidPhoneNumber(t *testing.T) {
	if err := validatePhoneNumber("2068773590"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}
}

func TestValidPhoneNumberWithExtension(t *testing.T) {
	if err := validatePhoneNumber("2068773590x123"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}

	if err := validatePhoneNumber("2068773590X123"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}
}

func TestInValidPhoneNumberShort(t *testing.T) {
	if err := validatePhoneNumber("206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInValidPhoneNumberAlpha(t *testing.T) {
	if err := validatePhoneNumber("a206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206877359a"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("2068a77359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206-877-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInValidPhoneNumberEmpty(t *testing.T) {
	if err := validatePhoneNumber(""); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInValidPhoneNumberExtensionInvalid(t *testing.T) {
	if err := validatePhoneNumber("2068773590x"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberInvalidAreaCode(t *testing.T) {
	if err := validatePhoneNumber("0008773590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}
