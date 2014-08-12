package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
)

type mediaUploadResponse struct {
	MediaID int64 `json:"media_id,string"`
}

func uploadMedia(t *testing.T, testData *TestData, accountID int64) int64 {
	store := storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "media")
	h := media.NewHandler(testData.DataApi, store)
	ts := httptest.NewServer(h)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", "example.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("Music")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	res, err := testData.AuthPost(ts.URL, writer.FormDataContentType(), body, accountID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	var r mediaUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}

	return r.MediaID
}

func TestMediaUpload(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	pr := SignupRandomTestPatient(t, testData)

	mediaID := uploadMedia(t, testData, pr.Patient.AccountId.Int64())

	store := storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "media")
	h := media.NewHandler(testData.DataApi, store)
	ts := httptest.NewServer(h)
	defer ts.Close()

	res, err := testData.AuthGet(fmt.Sprintf("%s?media_id=%d&claimer_type=&claimer_id=0", ts.URL, mediaID), pr.Patient.AccountId.Int64())

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
	if string(data) != "Musi" {
		t.Fatalf("Expected 'Musi'. Got '%s'.", string(data))
	}
}
