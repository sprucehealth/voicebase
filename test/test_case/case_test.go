package test_case

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseInfo_MessagingTPFlag(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	// treatment plan should be disabled given that the doctor has not yet been assigned to the case
	// messaging should be enables given that we let the patient message the care team at any point
	res, err := testData.AuthGet(testData.APIServer.URL+router.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	var responseData map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&responseData); err != nil {
		t.Fatal(err)
	}

	messagingEnabled := responseData["case_config"].(map[string]interface{})["messaging_enabled"].(bool)
	treatmentPlanEnabled := responseData["case_config"].(map[string]interface{})["treatment_plan_enabled"].(bool)

	test.Equals(t, true, messagingEnabled)
	test.Equals(t, false, treatmentPlanEnabled)

	// once the doctor submits the treatment plan both messaging and treatment plan should be enabled
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)
	res, err = testData.AuthGet(testData.APIServer.URL+router.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	messagingEnabled = responseData["case_config"].(map[string]interface{})["messaging_enabled"].(bool)
	treatmentPlanEnabled = responseData["case_config"].(map[string]interface{})["treatment_plan_enabled"].(bool)
	test.Equals(t, true, messagingEnabled)
	test.Equals(t, true, treatmentPlanEnabled)

	// lets ensure the case where the doctor has sent the message to the patient to enable messaging but has not yet submitted a treamtent plan
	_, tp = test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err = testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)
	test_integration.PostCaseMessage(t, testData, doctor.AccountId.Int64(), &messages.PostMessageRequest{
		CaseID:  tp.PatientCaseId.Int64(),
		Message: "foo",
	})
	res, err = testData.AuthGet(testData.APIServer.URL+router.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	messagingEnabled = responseData["case_config"].(map[string]interface{})["messaging_enabled"].(bool)
	treatmentPlanEnabled = responseData["case_config"].(map[string]interface{})["treatment_plan_enabled"].(bool)
	test.Equals(t, true, messagingEnabled)
	test.Equals(t, false, treatmentPlanEnabled)

}
