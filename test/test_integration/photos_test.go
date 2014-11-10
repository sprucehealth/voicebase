package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/test"
)

type photoUploadResponse struct {
	PhotoID  int64  `json:"photo_id,string"`
	PhotoURL string `json:"photo_url"`
}

func uploadPhoto(t *testing.T, testData *TestData, accountID int64) (int64, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", "example.jpg")
	test.OK(t, err)
	if _, err := part.Write([]byte("Foo")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	res, err := testData.AuthPost(testData.APIServer.URL+router.PhotoURLPath, writer.FormDataContentType(), body, accountID)
	test.OK(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	var r photoUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	return r.PhotoID, r.PhotoURL
}

func TestPhotoUpload(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	_, photoURL := uploadPhoto(t, testData, pr.Patient.AccountId.Int64())

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
