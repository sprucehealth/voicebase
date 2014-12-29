package test_integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/test"
)

type photoUploadResponse struct {
	PhotoID  int64  `json:"photo_id,string"`
	PhotoURL string `json:"photo_url"`
}

func UploadPhoto(t *testing.T, testData *TestData, accountID int64) (int64, string) {
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

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PhotoURLPath, writer.FormDataContentType(), body, accountID)
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
