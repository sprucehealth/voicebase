package test_case

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseUpdate_PresubmissionTriage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	pc, err := testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, true, pc.ClosedDate == nil)

	updatedStatus := common.PCStatusPreSubmissionTriage
	now := time.Now()
	timeoutDate := time.Now().Add(24 * time.Hour)
	test.OK(t, testData.DataAPI.UpdatePatientCase(tp.PatientCaseID.Int64(), &api.PatientCaseUpdate{
		Status:     &updatedStatus,
		ClosedDate: &now,
		TimeoutDate: api.NullableTime{
			Valid: true,
			Time:  &timeoutDate,
		},
	}))

	pc, err = testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, updatedStatus, pc.Status)
	test.Equals(t, true, pc.ClosedDate != nil)
	test.Equals(t, timeoutDate.Format("02-Jan-06 15:04"), pc.TimeoutDate.Format("02-Jan-06 15:04"))

	// now lets update the case to set the case to be triage_deleted
	updatedStatus = common.PCStatusPreSubmissionTriageDeleted
	test.OK(t, testData.DataAPI.UpdatePatientCase(tp.PatientCaseID.Int64(), &api.PatientCaseUpdate{
		Status: &updatedStatus,
		TimeoutDate: api.NullableTime{
			Valid: true,
		},
	}))
	pc, err = testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, updatedStatus, pc.Status)
	test.Equals(t, true, pc.TimeoutDate == nil)

}

func TestCase_TimedOut(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create multiple cases
	_, tp1 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	_, tp2 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	_, tp3 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// timeout 2 cases
	_, err = testData.DB.Exec(`UPDATE patient_case SET timeout_date = ? WHERE id in (?, ?)`, time.Now().Add(-time.Hour), tp1.PatientCaseID.Int64(), tp2.PatientCaseID.Int64())
	test.OK(t, err)

	// update 3rd case to timeout in the future
	_, err = testData.DB.Exec(`UPDATE patient_case SET timeout_date = ? WHERE id = ?`, time.Now().Add(time.Hour), tp3.PatientCaseID.Int64())

	// ensure that we get the two cases that have timed out
	cases, err := testData.DataAPI.TimedOutCases()
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	test.Equals(t, tp1.PatientCaseID.Int64(), cases[0].ID.Int64())
	test.Equals(t, tp2.PatientCaseID.Int64(), cases[1].ID.Int64())
}

// This test is to ensure that a case transitions from open to active upon submission
func TestCase_OpenToActiveTransition(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.ID.Int64())
	pv, err := pc.CreatePatientVisit(api.AcnePathwayTag, 0, test_integration.SetupTestHeaders())
	test.OK(t, err)
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, patientCase.Status, common.PCStatusOpen)

	test_integration.AddTestAddressForPatient(pr.Patient.ID.Int64(), testData, t)
	test_integration.AddTestPharmacyForPatient(pr.Patient.ID.Int64(), testData, t)
	test.OK(t, pc.SubmitPatientVisit(pv.PatientVisitID))

	patientCase, err = testData.DataAPI.GetPatientCaseFromID(patientCase.ID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PCStatusActive, patientCase.Status)
}

// This test is to ensure that the filtering of cases at the data layer
// works as expected
func TestCase_Filtering(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := test_integration.SignupRandomTestPatient(t, testData)
	patientID := pr.Patient.ID.Int64()

	// insert a few cases in different states
	_, err := testData.DB.Exec(`
		INSERT INTO patient_case (patient_id, status, name, clinical_pathway_id) 
		VALUES (?,?,'test', 1), (?,?, 'test', 1),(?,?,'test',1), (?,?, 'test',1), (?,?,'test',1)`,
		patientID, common.PCStatusActive.String(),
		patientID, common.PCStatusOpen.String(),
		patientID, common.PCStatusOpen.String(),
		patientID, common.PCStatusInactive.String(),
		patientID, common.PCStatusDeleted.String())
	test.OK(t, err)

	// now attempt to get cases with no parameters
	cases, err := testData.DataAPI.GetCasesForPatient(patientID, nil)
	test.OK(t, err)
	test.Equals(t, 4, len(cases))
	for _, pCase := range cases {
		test.Equals(t, false, pCase.Status == common.PCStatusDeleted)
	}

	cases, err = testData.DataAPI.GetCasesForPatient(patientID, []string{common.PCStatusOpen.String()})
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	test.Equals(t, common.PCStatusOpen, cases[0].Status)
	test.Equals(t, common.PCStatusOpen, cases[1].Status)

	cases, err = testData.DataAPI.GetCasesForPatient(patientID, []string{common.PCStatusInactive.String(), common.PCStatusActive.String()})
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
}

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
		Case *responses.Case `json:"case"`
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
	_, err = testData.DB.Exec(`UPDATE patient_case SET status = ? WHERE id = ?`,
		common.PCStatusUnsuitable.String(), tp.PatientCaseID.Int64())
	test.OK(t, err)
	res, err = testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, "Unsuitable for Spruce", responseData.Case.Diagnosis)

}
