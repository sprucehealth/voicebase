package apiservice

import "testing"

func TestValidZipcode(t *testing.T) {
	if err := validateZipcodeLocally("94115"); err != nil {
		t.Fatal("Expected zipcode to be valid")
	}

	if err := validateZipcodeLocally("94115-4161"); err != nil {
		t.Fatal("Expected zipcode to be valid")
	}

	if err := validateZipcodeLocally("941154161"); err != nil {
		t.Fatal("Expected zipcode to be valid")
	}

	if err := validateZipcodeLocally("041154161"); err != nil {
		t.Fatal("Expected zipcode to be valid")
	}
}

func TestInvalidZipcode(t *testing.T) {
	if err := validateZipcodeLocally("4115"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("94115x4161"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("941151231341"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("941151"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("94115-"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally(""); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("94115-123a"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("94115-12345"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}

	if err := validateZipcodeLocally("94115-1235a"); err == nil {
		t.Fatal("Expected zipcode to be invalid")
	}
}
