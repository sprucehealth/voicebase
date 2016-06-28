package test_patient_visit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice/apipaths"
	"github.com/sprucehealth/backend/cmd/svc/restapi/test/test_integration"
	"github.com/sprucehealth/backend/libs/test"
)

func TestPatientVisitMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	// save patient visit message
	msg := "nwcwg"
	jsonData, err := json.Marshal(map[string]interface{}{
		"visit_id": strconv.FormatInt(pv.PatientVisitID, 10),
		"message":  msg,
	})
	test.OK(t, err)
	res, err := testData.AuthPut(testData.APIServer.URL+apipaths.PatientVisitMessageURLPath, "application/json", bytes.NewReader(jsonData), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// get patient visit message
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientVisitMessageURLPath+"?visit_id="+strconv.FormatInt(pv.PatientVisitID, 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	var responseData struct {
		Message string `json:"message"`
	}
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, msg, responseData.Message)

}