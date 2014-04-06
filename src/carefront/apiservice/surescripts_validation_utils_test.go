package apiservice

import (
	"carefront/encoding"
	"testing"
)

func TestValidPhoneNumber(t *testing.T) {
	if err := validatePhoneNumber("2068773590"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}

	if err := validatePhoneNumber("206-877-3590"); err != nil {
		t.Fatal("Expected phone number to be invalid")
	}

}

func TestValidPhoneNumberWithExtension(t *testing.T) {
	if err := validatePhoneNumber("2068773590x123"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}

	if err := validatePhoneNumber("2068773590X123"); err != nil {
		t.Fatalf("Expected phone number to be valid: %+v", err)
	}

	if err := validatePhoneNumber("206-877-3590x12345135315151"); err != nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206-877-3590X1243"); err != nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberShort(t *testing.T) {
	if err := validatePhoneNumber("206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberAlpha(t *testing.T) {
	if err := validatePhoneNumber("a206877359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206877359a"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("2068a77359"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206-a12-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206-123-3590xa24"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("a06-123-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberLength(t *testing.T) {

	if err := validatePhoneNumber("206-1243-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("2064-123-3590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}

	if err := validatePhoneNumber("206-123-35904"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberEmpty(t *testing.T) {
	if err := validatePhoneNumber(""); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberExtensionInvalid(t *testing.T) {
	if err := validatePhoneNumber("2068773590x"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestInvalidPhoneNumberInvalidAreaCode(t *testing.T) {
	if err := validatePhoneNumber("0008773590"); err == nil {
		t.Fatal("Expected phone number to be invalid")
	}
}

func TestAgeCalculation(t *testing.T) {
	dob := encoding.Dob{
		Year:  2014,
		Month: 1,
		Day:   1,
	}

	if is18YearsOfAge(dob) {
		t.Fatal("Expected the age to be < 18 years")
	}

	dob.Year = 1995
	dob.Month = 1
	dob.Day = 1

	if !is18YearsOfAge(dob) {
		t.Fatal("Expected the age to be > 18 years")
	}

	dob.Year = 1997
	if is18YearsOfAge(dob) {
		t.Fatal("Expecte age to be < 18 years")
	}

}
