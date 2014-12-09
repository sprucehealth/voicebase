package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	patientpkg "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestIntake_PrefillQuestions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	testData.StartAPIServer(t)
	defer testData.Close()

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(pr.Patient.PatientID.Int64())
	test.OK(t, err)

	// answer the allergy question with a specific answer
	allergyQuestion, err := testData.DataAPI.GetQuestionInfo("q_allergic_medication_entry", api.EN_LANGUAGE_ID)
	test.OK(t, err)

	answerText := "Sulfa Drugs"
	specificAnswers := map[int64]*apiservice.AnswerToQuestionItem{
		allergyQuestion.QuestionID: &apiservice.AnswerToQuestionItem{
			QuestionID: allergyQuestion.QuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					AnswerText: answerText,
				},
			},
		},
	}

	rb := test_integration.PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(pv.PatientVisitID, pv.ClientLayout, specificAnswers, t)
	test_integration.SubmitAnswersIntakeForPatient(
		patient.PatientID.Int64(),
		patient.AccountID.Int64(),
		rb, testData, t)
	test_integration.SubmitPatientVisitForPatient(
		patient.PatientID.Int64(),
		pv.PatientVisitID,
		testData, t)

	// get the doctor to diagnose and submit the visit back to the patient
	visit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, visit.PatientCaseID.Int64())
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)
	test_integration.SubmitPatientVisitDiagnosis(pv.PatientVisitID, doctor, testData, t)
	tp := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   visit.PatientVisitID,
		ParentType: common.TPParentTypePatientVisit,
	}, nil, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// upload the intended followup layout that contains the questions to prefill
	test_integration.UploadIntakeLayoutConfiguration(&test_integration.UploadLayoutConfig{
		IntakeFileName:     "followup-intake-2-0-0.json",
		IntakeFileLocation: "../../info_intake/medhx-followup-intake-test.json",
		ReviewFileName:     "followup-review-2-0-0.json",
		ReviewFileLocation: "../../info_intake/medhx-followup-review-test.json",
		PatientAppVersion:  "1.2.0",
		DoctorAppVersion:   "1.2.0",
		Platform:           "iOS",
	}, testData, t)

	// now get the patient to start a followup visit
	followupVisit, visitLayout := createFollowupAndGetVisitLayout(patient, testData, t)
	followupVisitID := followupVisit.PatientVisitID.Int64()

	// the followup visit layout should contain the patient's
	// previous response to the allergy question given that it
	// was indicated to be prefilled with the response
	answers := visitLayout.Answers()
	questions := visitLayout.Questions()
	test.Equals(t, 1, len(answers[allergyQuestion.QuestionID]))
	answerIntake := answers[allergyQuestion.QuestionID][0].(*common.AnswerIntake)
	test.Equals(t, answerText, answerIntake.AnswerText)
	// ensure that the answer was marked as being prefilled at the question level
	for _, question := range questions {
		if question.QuestionTag == allergyQuestion.QuestionTag {
			test.Equals(t, true, question.PrefilledWithPreviousAnswers)
			break
		}
	}

	// now lets go ahead and submit answers for the followup visit
	// with updated answers for the allergy question so as to ensure that
	// the answers show up as updated for subsequent followup visits
	answerText = "Penicillins"
	specificAnswers[allergyQuestion.QuestionID].AnswerIntakes[0].AnswerText = answerText
	rb = test_integration.PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(
		followupVisitID,
		visitLayout, specificAnswers, t)
	test_integration.SubmitAnswersIntakeForPatient(
		patient.PatientID.Int64(),
		patient.AccountID.Int64(),
		rb, testData, t)
	test_integration.SubmitPatientVisitForPatient(
		patient.PatientID.Int64(),
		followupVisitID,
		testData, t)

	// now lets go ahead and have the doctor diagnose the visit and submit it back
	// to the patient
	test_integration.StartReviewingPatientVisit(followupVisitID, doctor, testData, t)
	test_integration.SubmitPatientVisitDiagnosis(followupVisitID, doctor, testData, t)
	tp = test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.TreatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// lets go ahead and generate another followup
	followupVisit, visitLayout = createFollowupAndGetVisitLayout(patient, testData, t)

	// the followup visit layout should contain the patient's
	// previous response to the allergy question given that it
	// was indicated to be prefilled with the response
	answers = visitLayout.Answers()
	questions = visitLayout.Questions()
	test.Equals(t, 1, len(answers[allergyQuestion.QuestionID]))
	answerIntake = answers[allergyQuestion.QuestionID][0].(*common.AnswerIntake)
	test.Equals(t, answerText, answerIntake.AnswerText)
	// ensure that the answer was marked as being prefilled at the question level
	for _, question := range questions {
		if question.QuestionTag == allergyQuestion.QuestionTag {
			test.Equals(t, true, question.PrefilledWithPreviousAnswers)
			break
		}
	}
}

func createFollowupAndGetVisitLayout(patient *common.Patient, testData *test_integration.TestData, t *testing.T) (*common.PatientVisit, *info_intake.InfoIntakeLayout) {
	_, err := patientpkg.CreatePendingFollowup(
		patient,
		testData.DataAPI,
		testData.AuthAPI,
		testData.Config.Dispatcher,
		testData.Config.Stores["media"],
		testData.Config.AuthTokenExpiration)
	test.OK(t, err)

	followupVisit, err := testData.DataAPI.GetLastCreatedPatientVisit(patient.PatientID.Int64())
	test.OK(t, err)
	followupVisitID := followupVisit.PatientVisitID.Int64()
	// indicate the followup visit to be in the open state as that
	// is the state the user would find the visit in if they were to
	// start the followup visit
	open := common.PVStatusOpen
	err = testData.DataAPI.UpdatePatientVisit(
		followupVisitID,
		&api.PatientVisitUpdate{
			Status: &open,
		})
	test.OK(t, err)
	followupVisit.Status = open

	// get the followup visit layout populated with any patient answers
	visitLayout, err := patientpkg.IntakeLayoutForVisit(
		testData.DataAPI,
		testData.Config.Stores["media"],
		testData.Config.AuthTokenExpiration,
		followupVisit)
	test.OK(t, err)

	return followupVisit, visitLayout
}
