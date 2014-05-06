package integration

import (
	"bytes"
	"carefront/photos"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type photoUploadResponse struct {
	PhotoID int64 `json:"photo_id"`
}

func uploadPhoto(t *testing.T, testData TestData, accountID int64) int64 {
	h := photos.NewHandler(testData.DataApi, testData.AWSAuth, "dev-carefront-test", "us-east-1")
	ts := httptest.NewServer(h)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", "example.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("Foo")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	res, err := authPost(ts.URL, writer.FormDataContentType(), body, accountID)
	if err != nil {
		t.Fatal(err)
	}
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
	testData := setupIntegrationTest(t)
	defer tearDownIntegrationTest(t, testData)

	pr := signupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	photoID := uploadPhoto(t, testData, pr.Patient.AccountId.Int64())

	h := photos.NewHandler(testData.DataApi, testData.AWSAuth, "dev-carefront-test", "us-east-1")
	ts := httptest.NewServer(h)
	defer ts.Close()

	res, err := authGet(fmt.Sprintf("%s?photo_id=%d&claimer_type=&claimer_id=0", ts.URL, photoID), pr.Patient.AccountId.Int64())
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Foo" {
		t.Fatalf("Expected 'Foo'. Got '%s'.", string(data))
	}
}
