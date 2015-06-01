package test_followup

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFollowup_Diagnose(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create and submit visit for patient\
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	initialVisitID := pv.PatientVisitID
	patient, err := testData.DataAPI.GetPatientFromID(tp.PatientID)
	test.OK(t, err)
	patientID := patient.ID.Int64()
	patientAccountID := patient.AccountID.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// now lets treat the initial visit with a known diagnosis
	diagnosisQuestionID := test_integration.GetQuestionIDForQuestionTag("q_acne_diagnosis", 1, testData, t)
	acneTypeQuestionID := test_integration.GetQuestionIDForQuestionTag("q_acne_type", 1, testData, t)

	diagnosisIntakeRequestData := test_integration.SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionID:  []string{"a_acne_inflammatory"},
	}, pv.PatientVisitID, testData, t)

	test_integration.SubmitPatientVisitDiagnosisWithIntake(pv.PatientVisitID,
		doctor.AccountID.Int64(), diagnosisIntakeRequestData, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	pCase, err := testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)

	// now lets try to create a followup visit
	_, err = patientpkg.CreatePendingFollowup(patient, pCase, testData.DataAPI, testData.AuthAPI, testData.Config.Dispatcher)
	test.OK(t, err)

	followupVisit, err := testData.DataAPI.GetPatientVisitForSKU(patientID, test_integration.SKUAcneFollowup)
	test.OK(t, err)

	// query for the followup
	pv = test_integration.QueryPatientVisit(
		followupVisit.PatientVisitID.Int64(),
		patientAccountID,
		map[string]string{
			"S-Version": "Patient;Test;1.0.0;0001",
			"S-OS":      "iOS;7.1",
			"S-Device":  "Phone;iPhone6,1;640;1136;2.0",
		},
		testData,
		t)

	// lets have the patient submit the followup
	patientIntakeRequestData := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout, t)
	test_integration.SubmitAnswersIntakeForPatient(patientID, patientAccountID, patientIntakeRequestData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientID, followupVisit.PatientVisitID.Int64(), testData, t)

	// lets have the doctor start reviewing the visit
	test_integration.StartReviewingPatientVisit(followupVisit.PatientVisitID.Int64(), doctor, testData, t)

	// now lets query for the diagnosis without having submitted anything.
	// at this point the doctor should get back the diagnosis of the previous visit
	diagnosisLayout, err := patient_visit.GetDiagnosisLayout(testData.DataAPI, followupVisit, doctor.ID.Int64())
	test.OK(t, err)

	// ensure that the diagnosis layout has all the answers populated from the previous diagnosis
	test_integration.CompareDiagnosisWithDoctorIntake(diagnosisIntakeRequestData, diagnosisLayout, testData, t)

	// now lets update the diagnosis for the followup and ensure that it updates the follow up visit
	// diagnosis as well as the case diagnosis
	diagnosisIntakeRequestData = test_integration.SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionID: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionID:  []string{"a_acne_comedonal"},
	}, followupVisit.PatientVisitID.Int64(), testData, t)

	// lets submit the diagnosis for followup
	test_integration.SubmitPatientVisitDiagnosisWithIntake(followupVisit.PatientVisitID.Int64(),
		doctor.AccountID.Int64(), diagnosisIntakeRequestData, testData, t)

	// now lets query for the diagnosis of the visit to ensure that its returned as expected
	initialVisitDiagnosis, err := testData.DataAPI.DiagnosisForVisit(initialVisitID)
	test.OK(t, err)
	test.Equals(t, "Inflammatory Acne", initialVisitDiagnosis)

	followupVisitDiagnosis, err := testData.DataAPI.DiagnosisForVisit(followupVisit.PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, "Comedonal Acne", followupVisitDiagnosis)

	// lets start a treatment plan and submit it back to the patient
	tp2 := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	test_integration.SubmitPatientVisitBackToPatient(tp2.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// diagnosis for case should indicate the latest diagnosis
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseID.Int64(), 10), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	var responseData struct {
		Case *responses.Case `json:"case"`
	}
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, followupVisitDiagnosis, responseData.Case.Diagnosis)

}
