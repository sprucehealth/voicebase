package test_integration

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPhotoUpload(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	_, photoURL := UploadPhoto(t, testData, pr.Patient.AccountID.Int64())

	// TODO: this is a hack to replace the domain in the mediaURL. There's no easy way
	// to have it be correct since the MediaStore needs to be configured before the server
	// is started, but the :port of the test server isn't known until after it's started.
	ur, err := url.Parse(testData.APIServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	photoURL = strings.Replace(photoURL, "example.com", ur.Host, -1)

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
