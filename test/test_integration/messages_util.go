package test_integration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
)

func PostCaseMessage(t *testing.T, testData *TestData, accountID int64, req *messages.PostMessageRequest) int64 {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		t.Fatal(err)
	}
	res, err := testData.AuthPost(testData.APIServer.URL+router.CaseMessagesURLPath, "application/json", body, accountID)
	test.OK(t, err)
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
