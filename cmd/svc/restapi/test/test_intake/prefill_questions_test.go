package test_intake

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/diagnosis/handlers"
	"github.com/sprucehealth/backend/cmd/svc/restapi/info_intake"
	patientpkg "github.com/sprucehealth/backend/cmd/svc/restapi/patient"
	"github.com/sprucehealth/backend/cmd/svc/restapi/test/test_integration"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
)

func TestIntake_PrefillQuestions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	testData.StartAPIServer(t)
	defer testData.Close(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	doctorCLI := test_integration.DoctorClient(testData, t, dr.DoctorID)
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)
	pv := test_integration.CreatePatientVisitForPatient(pr.Patient.ID, testData, t)

	patient, err := testData.DataAPI.GetPatientFromID(pr.Patient.ID)
	test.OK(t, err)

	// answer the allergy question with a specific answer
	allergyQuestion, err := testData.DataAPI.GetQuestionInfo("q_allergic_medication_entry", api.LanguageIDEnglish, 1)
	test.OK(t, err)

	answerText := "Sulfa Drugs"
	specificAnswers := map[int64]*apiservice.QuestionAnswerItem{
		allergyQuestion.QuestionID: {
			QuestionID: allergyQuestion.QuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				{AnswerText: answerText},
			},
		},
	}

	rb := test_integration.PrepareAnswersForQuestionsWithSomeSpecifiedAnswers(pv.PatientVisitID, pv.ClientLayout.InfoIntakeLayout, specificAnswers, t)
	test_integration.SubmitAnswersIntakeForPatient(
		patient.ID,
		patient.AccountID.Int64(),
		rb, testData, t)
	test_integration.SubmitPatientVisitForPatient(
		patient.ID,
		pv.PatientVisitID,
		testData, t)

	// get the doctor to diagnose and submit the visit back to the patient
	visit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test_integration.GrantDoctorAccessToPatientCase(t, testData, doctor, visit.PatientCaseID.Int64())
	test_integration.StartReviewingPatientVisit(pv.PatientVisitID, doctor, testData, t)
	test.OK(t, doctorCLI.CreateDiagnosisSet(&handlers.DiagnosisListRequestData{
		VisitID: pv.PatientVisitID,
		Diagnoses: []*handlers.DiagnosisInputItem{
			{
				CodeID: "diag_l783",
			},
		},
	}))
	tp := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   visit.ID,
		ParentType: common.TPParentTypePatientVisit,
	}, nil, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// upload the intended followup layout that contains the questions to prefill
	test_integration.UploadIntakeLayoutConfiguration(&test_integration.UploadLayoutConfig{
		IntakeFileName:     "followup-intake-2-0-0.json",
		IntakeFileLocation: "../data/medhx-followup-intake-test.json",
		ReviewFileName:     "followup-review-2-0-0.json",
		ReviewFileLocation: "../data/medhx-followup-review-test.json",
		PatientAppVersion:  "1.2.0",
		DoctorAppVersion:   "1.2.0",
		Platform:           "iOS",
	}, testData, t)

	// now get the patient to start a followup visit
	followupVisit, visitLayout := createFollowupAndGetVisitLayout(patient, visit.PatientCaseID.Int64(), testData, t)
	followupVisitID := followupVisit.ID.Int64()
	test.Equals(t, true, followupVisitID != visit.ID.Int64())

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
		patient.ID,
		patient.AccountID.Int64(),
		rb, testData, t)
	test_integration.SubmitPatientVisitForPatient(
		patient.ID,
		followupVisitID,
		testData, t)

	// now lets go ahead and have the doctor diagnose the visit and submit it back
	// to the patient
	test_integration.StartReviewingPatientVisit(followupVisitID, doctor, testData, t)
	test.OK(t, doctorCLI.CreateDiagnosisSet(&handlers.DiagnosisListRequestData{
		VisitID: pv.PatientVisitID,
		Diagnoses: []*handlers.DiagnosisInputItem{
			{
				CodeID: "diag_l783",
			},
		},
	}))
	tp = test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.TreatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// lets go ahead and generate another followup
	followupVisit2, visitLayout := createFollowupAndGetVisitLayout(patient, tp.TreatmentPlan.PatientCaseID.Int64(), testData, t)
	test.Equals(t, true, followupVisit.ID.Int64() != followupVisit2.ID.Int64())

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

func createFollowupAndGetVisitLayout(patient *common.Patient, caseID int64, testData *test_integration.TestData, t *testing.T) (*common.PatientVisit, *info_intake.InfoIntakeLayout) {
	patientCase, err := testData.DataAPI.GetPatientCaseFromID(caseID)
	test.OK(t, err)

	_, err = patientpkg.CreatePendingFollowup(
		patient,
		patientCase,
		testData.DataAPI,
		testData.AuthAPI,
		testData.Config.Dispatcher)
	test.OK(t, err)

	visits, err := testData.DataAPI.GetVisitsForCase(caseID, nil)
	test.OK(t, err)

	sort.Sort(sort.Reverse(common.ByPatientVisitCreationDate(visits)))
	followupVisit := visits[0]
	test.Equals(t, test_integration.SKUAcneFollowup, followupVisit.SKUType)

	followupVisitID := followupVisit.ID.Int64()
	// indicate the followup visit to be in the open state as that
	// is the state the user would find the visit in if they were to
	// start the followup visit
	_, err = testData.DataAPI.UpdatePatientVisit(
		followupVisitID,
		&api.PatientVisitUpdate{
			Status: ptr.String(common.PVStatusOpen),
		})
	test.OK(t, err)
	followupVisit.Status = common.PVStatusOpen

	// get the followup visit layout populated with any patient answers
	intakeInfo, err := patientpkg.IntakeLayoutForVisit(
		testData.DataAPI,
		testData.Config.APIDomain,
		testData.Config.WebDomain,
		testData.Config.MediaStore,
		testData.Config.AuthTokenExpiration,
		followupVisit, nil, api.RoleDoctor)
	test.OK(t, err)

	return followupVisit, intakeInfo.ClientLayout.InfoIntakeLayout
}