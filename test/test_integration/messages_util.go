package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
)

func AssignCaseMessage(t *testing.T, testData *TestData, accountID int64, req *messages.PostMessageRequest) int64 {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		t.Fatal(err)
	}
	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorAssignCaseURLPath, "application/json", body, accountID)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	newConvRes := &messages.PostMessageResponse{}
	if err := json.NewDecoder(res.Body).Decode(newConvRes); err != nil {
		t.Fatal(err)
	}
	return newConvRes.MessageID
}
