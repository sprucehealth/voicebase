package test_patient_visit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientVisitMessage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)

	// save patient visit message
	msg := "nwcwg"
	jsonData, err := json.Marshal(map[string]interface{}{
		"visit_id": strconv.FormatInt(pv.PatientVisitId, 10),
		"message":  msg,
	})
	test.OK(t, err)
	res, err := testData.AuthPut(testData.APIServer.URL+router.PatientVisitMessageURLPath, "application/json", bytes.NewReader(jsonData), patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	// get patient visit message
	res, err = testData.AuthGet(testData.APIServer.URL+router.PatientVisitMessageURLPath+"?visit_id="+strconv.FormatInt(pv.PatientVisitId, 10), patient.AccountId.Int64())
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
