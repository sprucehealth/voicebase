package test_case

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseInfo_MessagingTPFlag(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	doctorCli := test_integration.DoctorClient(testData, t, dr.DoctorID)

	// treatment plan should be disabled given that the doctor has not yet been assigned to the case
	// messaging should be enables given that we let the patient message the care team at any point
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
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

	// lets submit a diagnosis for the patient
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitID, doctor, testData, t)

	// once the doctor submits the treatment plan both messaging and treatment plan should be enabled
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
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
	patient, err = testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)
	_, err = doctorCli.PostCaseMessage(tp.PatientCaseID.Int64(), "foo", nil)
	test.OK(t, err)
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
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

func TestCaseInfo_DiagnosisField(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	// diagnosis field should say Pending until the doctor has actually reviewed the case
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	var responseData struct {
		Case *common.PatientCase `json:"case"`
	}
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, "Pending", responseData.Case.Diagnosis)

	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitID, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)
	// diagnosis field should now be non empty and the same as the patient visit's diagnosis
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	diagnosis, err := testData.DataAPI.DiagnosisForVisit(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, diagnosis, responseData.Case.Diagnosis)

	// Now lets make sure that if the patient case is marked as unsuitable, the diagnosis type exposes the unsuitable status
	_, tp = test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err = testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	// lets go ahead and manually update the status of the case to be unsuitable because that is what we would do in the real world
	_, err = testData.DB.Exec(`update patient_case set status = ? where id = ?`, common.PCStatusUnsuitable, tp.PatientCaseID.Int64())
	test.OK(t, err)
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, "Unsuitable for Spruce", responseData.Case.Diagnosis)

}
