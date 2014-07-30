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

	"github.com/sprucehealth/backend/audio"
	"github.com/sprucehealth/backend/libs/storage"
)

type audioUploadResponse struct {
	AudioID int64 `json:"audio_id,string"`
}

func uploadAudio(t *testing.T, testData *TestData, accountID int64) int64 {
	store := storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "audio")
	h := audio.NewHandler(testData.DataApi, store)
	ts := httptest.NewServer(h)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", "example.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("Musi")); err != nil {
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
	var r audioUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}

	return r.AudioID
}

func TestAudioUpload(t *testing.T) {
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)
	pr := SignupRandomTestPatient(t, testData)

	audioID := uploadAudio(t, testData, pr.Patient.AccountId.Int64())

	store := storage.NewS3(testData.AWSAuth, "us-east-1", "test-spruce-storage", "audio")
	h := audio.NewHandler(testData.DataApi, store)
	ts := httptest.NewServer(h)
	defer ts.Close()

	res, err := testData.AuthGet(fmt.Sprintf("%s?audio_id=%d&claimer_type=&claimer_id=1", ts.URL, audioID), pr.Patient.AccountId.Int64())
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
