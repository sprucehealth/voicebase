package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/test"
)

type photoUploadResponse struct {
	PhotoID int64 `json:"photo_id,string"`
}

func uploadPhoto(t *testing.T, testData *TestData, accountID int64) int64 {
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

	return r.PhotoID
}

func TestPhotoUpload(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := SignupRandomTestPatient(t, testData)

	photoID := uploadPhoto(t, testData, pr.Patient.AccountId.Int64())

	res, err := testData.AuthGet(fmt.Sprintf("%s?photo_id=%d&claimer_type=&claimer_id=0", testData.APIServer.URL+router.PhotoURLPath, photoID), pr.Patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	test.OK(t, err)
	if string(data) != "Foo" {
		t.Fatalf("Expected 'Foo'. Got '%s'.", string(data))
	}
}
