package test_followup

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFollowup_Diagnose(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create and submit visit for patient\
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	initialVisitID := pv.PatientVisitId
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)
	patientID := patient.PatientId.Int64()
	patientAccountID := patient.AccountId.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// now lets treat the initial visit with a known diagnosis
	diagnosisQuestionId := test_integration.GetQuestionIdForQuestionTag("q_acne_diagnosis", testData, t)
	acneTypeQuestionId := test_integration.GetQuestionIdForQuestionTag("q_acne_type", testData, t)

	diagnosisIntakeRequestData := test_integration.SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_inflammatory"},
	}, pv.PatientVisitId, testData, t)

	test_integration.SubmitPatientVisitDiagnosisWithIntake(pv.PatientVisitId,
		doctor.AccountId.Int64(), diagnosisIntakeRequestData, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// lets wait for a moment so as to let a second elapse before creating the next followup
	// so that there is time between the creation of the initial visit and the followup
	time.Sleep(time.Second)

	// now lets try to create a followup visit
	_, err = patientpkg.CreatePendingFollowup(patient, testData.DataApi, testData.AuthApi,
		testData.Config.Dispatcher, testData.Config.Stores["media"],
		testData.Config.AuthTokenExpiration)
	test.OK(t, err)

	followupVisit, err := testData.DataApi.GetPatientVisitForSKU(patientID, sku.AcneFollowup)
	test.OK(t, err)

	// query for the followup
	pv = test_integration.QueryPatientVisit(
		followupVisit.PatientVisitId.Int64(),
		patientAccountID,
		map[string]string{
			"S-Version": "Patient;Test;1.0.0;0001",
			"S-OS":      "iOS;7.1",
			"S-Device":  "Phone;iPhone6,1;640;1136;2.0",
		},
		testData,
		t)

	// lets have the patient submit the followup
	patientIntakeRequestData := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(patientID, patientAccountID, patientIntakeRequestData, testData, t)
	test_integration.SubmitPatientVisitForPatient(patientID, followupVisit.PatientVisitId.Int64(), testData, t)

	// lets have the doctor start reviewing the visit
	test_integration.StartReviewingPatientVisit(followupVisit.PatientVisitId.Int64(), doctor, testData, t)

	// now lets query for the diagnosis without having submitted anything.
	// at this point the doctor should get back the diagnosis of the previous visit
	diagnosisLayout, err := patient_visit.GetDiagnosisLayout(testData.DataApi, followupVisit, doctor.DoctorId.Int64())
	test.OK(t, err)

	// ensure that the diagnosis layout has all the answers populated from the previous diagnosis
	test_integration.CompareDiagnosisWithDoctorIntake(diagnosisIntakeRequestData, diagnosisLayout, testData, t)

	// now lets update the diagnosis for the followup and ensure that it updates the follow up visit
	// diagnosis as well as the case diagnosis
	diagnosisIntakeRequestData = test_integration.SetupAnswerIntakeForDiagnosis(map[int64][]string{
		diagnosisQuestionId: []string{"a_doctor_acne_vulgaris"},
		acneTypeQuestionId:  []string{"a_acne_comedonal"},
	}, followupVisit.PatientVisitId.Int64(), testData, t)

	// lets submit the diagnosis for followup
	test_integration.SubmitPatientVisitDiagnosisWithIntake(followupVisit.PatientVisitId.Int64(),
		doctor.AccountId.Int64(), diagnosisIntakeRequestData, testData, t)

	// now lets query for the diagnosis of the visit to ensure that its returned as expected
	initialVisitDiagnosis, err := testData.DataApi.DiagnosisForVisit(initialVisitID)
	test.OK(t, err)
	test.Equals(t, "Inflammatory Acne", initialVisitDiagnosis)

	followupVisitDiagnosis, err := testData.DataApi.DiagnosisForVisit(followupVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, "Comedonal Acne", followupVisitDiagnosis)

	// lets start a treatment plan and submit it back to the patient
	tp2 := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   tp.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	test_integration.SubmitPatientVisitBackToPatient(tp2.TreatmentPlan.Id.Int64(), doctor, testData, t)

	// diagnosis for case should indicate the latest diagnosis
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.PatientCasesURLPath+"?case_id="+strconv.FormatInt(tp.PatientCaseId.Int64(), 10), patient.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	var responseData struct {
		Case *common.PatientCase `json:"case"`
	}
	err = json.NewDecoder(res.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, followupVisitDiagnosis, responseData.Case.Diagnosis)

}
