package test_integration

import (
	"bytes"
	"github.com/sprucehealth/backend/messages"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func PostCaseMessage(t *testing.T, testData *TestData, accountID int64, req *messages.PostMessageRequest) int64 {
	doctorConvoServer := httptest.NewServer(messages.NewHandler(testData.DataApi))
	defer doctorConvoServer.Close()

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		t.Fatal(err)
	}
	res, err := testData.AuthPost(doctorConvoServer.URL, "application/json", body, accountID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("Expected status 200. Got %d", res.StatusCode)
	}
	newConvRes := &messages.PostMessageResponse{}
	if err := json.NewDecoder(res.Body).Decode(newConvRes); err != nil {
		t.Fatal(err)
	}
	return newConvRes.MessageID
}
