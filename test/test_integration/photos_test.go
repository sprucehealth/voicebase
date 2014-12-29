package test_integration

import (
	"io/ioutil"
	"net/http"
	"testing"
)

func TestPhotoUpload(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	_, photoURL := UploadPhoto(t, testData, pr.Patient.AccountID.Int64())

	linkData, err := http.Get(photoURL)
	defer linkData.Body.Close()

	if err != nil {
		t.Fatal(err)
	}
	fileContents, err := ioutil.ReadAll(linkData.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(fileContents) != "Foo" {
		t.Fatalf("Expected 'Foo'. Got '%s'.", string(fileContents))
	}
}
