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

type mediaUploadResponse struct {
	MediaID int64 `json:"media_id,string"`
}

type mediaResponse struct {
	MediaType string `json:"media_type, required"`
	MediaURL  string `json:"media_url, required"`
}

func uploadMedia(t *testing.T, testData *TestData, accountID int64) int64 {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", "example.mp4")
	test.OK(t, err)
	if _, err := part.Write([]byte("Music")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	res, err := testData.AuthPost(testData.APIServer.URL+router.MediaURLPath, writer.FormDataContentType(), body, accountID)
	test.OK(t, err)
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
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := SignupRandomTestPatient(t, testData)

	mediaID := uploadMedia(t, testData, pr.Patient.AccountId.Int64())

	res, err := testData.AuthGet(fmt.Sprintf("%s?media_id=%d&claimer_type=&claimer_id=0", testData.APIServer.URL+router.MediaURLPath, mediaID), pr.Patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200. Got %d", res.StatusCode)
	}
	var resp mediaResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	linkData, err := http.Get(resp.MediaURL)
	if err != nil {
		t.Fatal(err)
	}
	fileContents, err := ioutil.ReadAll(linkData.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(fileContents) != "Music" {
		t.Fatalf("Expected 'Music'. Got '%s'.", string(fileContents))
	}
}
