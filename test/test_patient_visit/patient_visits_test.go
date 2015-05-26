package test_patient_visit

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientVisitsList_Patient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// lets create and submit a visit
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	patientCase, err := testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)
	verifyVisitSubmissionTags(t, testData, patientCase, true, true)

	// ensure that the list returns 1 visit
	res, err := getPatientVisits(patient.AccountID.Int64(), tp.PatientCaseID.Int64(), false, t, testData)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	var response map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	test.Equals(t, 1, len(response["visits"].([]interface{})))

	// lets get doctor to submit the visit back to patient
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// now lets start a followup visit
	err = test_integration.CreateFollowupVisitForPatient(patient, patientCase, t, testData)
	test.OK(t, err)

	// now lets query for visits again and ensure that we have 2 visits returned
	res, err = getPatientVisits(patient.AccountID.Int64(), tp.PatientCaseID.Int64(), false, t, testData)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	response = make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	test.Equals(t, 2, len(response["visits"].([]interface{})))

	// lets try to run the same query again with completed set to true to ensure that just 1 visit is returned
	res, err = getPatientVisits(patient.AccountID.Int64(), tp.PatientCaseID.Int64(), true, t, testData)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	response = make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	test.Equals(t, 1, len(response["visits"].([]interface{})))

	patientVisit, err := testData.DataAPI.GetPatientVisitForSKU(patient.ID.Int64(), test_integration.SKUAcneFollowup)
	test.OK(t, err)

	test_integration.SubmitPatientVisitForPatient(patient.ID.Int64(), patientVisit.PatientVisitID.Int64(), testData, t)

	// now query to ensure that 2 visits are returned when completed is true
	res, err = getPatientVisits(patient.AccountID.Int64(), tp.PatientCaseID.Int64(), true, t, testData)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	response = make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	test.Equals(t, 2, len(response["visits"].([]interface{})))
}

func verifyVisitSubmissionTags(t *testing.T, testData *test_integration.TestData, pCase *common.PatientCase, newPatient, initialVisit bool) {
	var count int
	if err := testData.DB.QueryRow("SELECT COUNT(*) FROM tag_membership WHERE case_id = ?", pCase.ID.Int64()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	test.Equals(t, 10, count)
}

func TestQueryingSubmittedVisits(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// lets create and submit a visit
	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)

	visits, err := testData.DataAPI.VisitsSubmittedForPatientSince(patient.ID.Int64(), time.Now().Add(10*time.Minute))
	test.OK(t, err)
	test.Equals(t, 0, len(visits))

	visits, err = testData.DataAPI.VisitsSubmittedForPatientSince(patient.ID.Int64(), time.Now().Add(-10*time.Minute))
	test.OK(t, err)
	test.Equals(t, 1, len(visits))
}

func getPatientVisits(patientAccountID, patientCaseID int64, completed bool, t *testing.T, testData *test_integration.TestData) (*http.Response, error) {
	params := url.Values{}
	params.Set("case_id", strconv.FormatInt(patientCaseID, 10))
	params.Set("completed", strconv.FormatBool(completed))

	req, err := http.NewRequest("GET", testData.APIServer.URL+apipaths.PatientVisitsListURLPath+"?"+params.Encode(), nil)
	req.Header.Set("S-Version", "Patient;Test;1.0.0;0001")
	req.Header.Set("S-OS", "iOS;7.1")
	req.Header.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	token, err := testData.AuthAPI.GetToken(patientAccountID)
	test.OK(t, err)
	req.Header.Set("Authorization", "token "+token)
	return http.DefaultClient.Do(req)
}
