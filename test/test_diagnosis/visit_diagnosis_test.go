package test_diagnosis

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	diaghandlers "github.com/sprucehealth/backend/diagnosis/handlers"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDiagnosisSet(t *testing.T) {
	diagnosisService := setupDiagnosisService(t)

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.DiagnosisAPI = diagnosisService
	testData.AdminConfig.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	// add a couple diagnosis for testing purposes
	codeID1 := "diag_l710"
	codeID2 := "diag_l719"

	diagnosisMap, err := diagnosisService.DiagnosisForCodeIDs([]string{codeID1, codeID2})
	test.OK(t, err)
	d1 := diagnosisMap[codeID1]

	admin := test_integration.CreateRandomAdmin(t, testData)

	test_integration.UploadDetailsLayoutForDiagnosis(`
	{
	"diagnosis_layouts" : [
	{
		"code_id" : "diag_l710",
		"layout_version" : "1.0.0",
		"questions" : [
		{
			"question" : "q_acne_severity",
			"additional_fields": {
      			 "style": "brief_title"
      		}
		},
		{
			"question" : "q_acne_type"
		}]
	}
	]
	}`, admin.AccountID.Int64(), testData, t)

	// lets get the questionID and answerIDs of the questions
	questionInfos, err := testData.DataAPI.GetQuestionInfoForTags([]string{"q_acne_severity", "q_acne_type"}, api.LanguageIDEnglish)
	test.OK(t, err)

	answerInfos, err := testData.DataAPI.GetAnswerInfoForTags([]string{"a_doctor_acne_severity_moderate", "a_acne_comedonal"}, api.LanguageIDEnglish)
	test.OK(t, err)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create diagnosis set including both diagnosis codes
	answers := []*apiservice.QuestionAnswerItem{
		{
			QuestionID: questionInfos[0].QuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerID: answerInfos[0].AnswerID,
				},
			},
		},
		{
			QuestionID: questionInfos[1].QuestionID,
			AnswerIntakes: []*apiservice.AnswerItem{
				&apiservice.AnswerItem{
					PotentialAnswerID: answerInfos[1].AnswerID,
				},
			},
		},
	}

	note := "testing w/ this note"
	err = doctorClient.CreateDiagnosisSet(&diaghandlers.DiagnosisListRequestData{
		VisitID:      pv.PatientVisitID,
		InternalNote: note,
		Diagnoses: []*diaghandlers.DiagnosisInputItem{
			{
				CodeID: codeID1,
				LayoutVersion: &common.Version{
					Major: 1,
					Minor: 0,
					Patch: 0,
				},
				Answers: answers,
			},
			{
				CodeID: codeID2,
			},
		},
	})
	test.OK(t, err)

	// get the diagnosis to test that it was set as expected
	diagnosisListResponse, err := doctorClient.ListDiagnosis(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, 2, len(diagnosisListResponse.Diagnoses))
	test.Equals(t, 2, len(diagnosisListResponse.Diagnoses[0].Questions))
	test.Equals(t, note, diagnosisListResponse.Notes)
	test.Equals(t, false, diagnosisListResponse.CaseManagement.Unsuitable)
	test.Equals(t, d1.Description, diagnosisListResponse.Diagnoses[0].Title)
	test.Equals(t, codeID1, diagnosisListResponse.Diagnoses[0].CodeID)
	test.Equals(t, codeID2, diagnosisListResponse.Diagnoses[1].CodeID)
	test.Equals(t, true, diagnosisListResponse.Diagnoses[0].HasDetails)
	test.Equals(t, false, diagnosisListResponse.Diagnoses[1].HasDetails)
	test.Equals(t, "1.0.0", diagnosisListResponse.Diagnoses[0].LayoutVersion.String())
	test.Equals(t, "1.0.0", diagnosisListResponse.Diagnoses[0].LatestLayoutVersion.String())
	test.Equals(t, len(answers), len(diagnosisListResponse.Diagnoses[0].Answers))

	// populate a mapping of question to answer to ensure that each question have been answered with
	// the expected set of answers
	answersToQuestions := make(map[int64]*apiservice.QuestionAnswerItem)
	for _, qaItem := range diagnosisListResponse.Diagnoses[0].Answers {
		answersToQuestions[qaItem.QuestionID] = qaItem
	}
	test.Equals(t, answers[0], answersToQuestions[questionInfos[0].QuestionID])
	test.Equals(t, answers[1], answersToQuestions[questionInfos[1].QuestionID])

	// lets update the diagnosis set to remove one code and the note as well
	note = "updated note"
	err = doctorClient.CreateDiagnosisSet(&diaghandlers.DiagnosisListRequestData{
		VisitID:      pv.PatientVisitID,
		InternalNote: note,
		Diagnoses: []*diaghandlers.DiagnosisInputItem{
			{
				CodeID: codeID1,
				LayoutVersion: &common.Version{
					Major: 1,
					Minor: 0,
					Patch: 0,
				},
				Answers: answers,
			},
		},
	})
	test.OK(t, err)

	diagnosisListResponse, err = doctorClient.ListDiagnosis(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, 1, len(diagnosisListResponse.Diagnoses))
	test.Equals(t, codeID1, diagnosisListResponse.Diagnoses[0].CodeID)
	test.Equals(t, note, diagnosisListResponse.Notes)

	// now lets update the layout for code1 and ensure that the latest layout version is updated
	// to indicate the new one
	test_integration.UploadDetailsLayoutForDiagnosis(`
	{
	"diagnosis_layouts" : [
	{
		"code_id" : "diag_l710",
		"layout_version" : "1.1.0",
		"questions" : [
		{
			"question" : "q_acne_severity",
			"additional_fields": {
      			 "style": "brief_title"
      		}
		}]
	}
	]
	}`, admin.AccountID.Int64(), testData, t)

	diagnosisListResponse, err = doctorClient.ListDiagnosis(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, "1.1.0", diagnosisListResponse.Diagnoses[0].LatestLayoutVersion.String())
	test.Equals(t, "1.0.0", diagnosisListResponse.Diagnoses[0].LayoutVersion.String())

}

// TestDiagnosisSet_Followup is an integration test that ensures that
// the diagnosis set from a previous visit is populated for a followup visit
// when there is no active diagnosis set for the followup visit
func TestDiagnosisSet_Followup(t *testing.T) {
	diagnosisService := setupDiagnosisService(t)

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	pCase, err := testData.DataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
	test.OK(t, err)

	note := "testing w/ this note"
	err = doctorClient.CreateDiagnosisSet(&diaghandlers.DiagnosisListRequestData{
		VisitID:      pv.PatientVisitID,
		InternalNote: note,
		Diagnoses: []*diaghandlers.DiagnosisInputItem{
			{
				CodeID: "diag_l710",
			},
			{
				CodeID: "diag_l719",
			},
		},
	})
	test.OK(t, err)

	test.OK(t, doctorClient.UpdateTreatmentPlanNote(tp.ID.Int64(), "fo"))
	test.OK(t, doctorClient.SubmitTreatmentPlan(tp.ID.Int64()))

	// Create a followup visit
	patient, err := testData.DataAPI.Patient(tp.PatientID, true)
	err = test_integration.CreateFollowupVisitForPatient(patient, pCase, t, testData)
	test.OK(t, err)

	visits, err := testData.DataAPI.GetVisitsForCase(tp.PatientCaseID.Int64(), nil)
	sort.Sort(sort.Reverse(common.ByPatientVisitCreationDate(visits)))

	// the diagnosis for the followup visit should match the diagnosis created for the
	// initial visit
	test.OK(t, err)
	diagnosisListResponse, err := doctorClient.ListDiagnosis(visits[0].PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, true, visits[0].IsFollowup)
	test.Equals(t, 2, len(diagnosisListResponse.Diagnoses))
	test.Equals(t, note, diagnosisListResponse.Notes)
	test.Equals(t, false, diagnosisListResponse.CaseManagement.Unsuitable)
	test.Equals(t, "diag_l710", diagnosisListResponse.Diagnoses[0].CodeID)
	test.Equals(t, "diag_l719", diagnosisListResponse.Diagnoses[1].CodeID)

	// now lets go ahead and add a specific diagnosis fro for the followup and ensure that
	// the diagnosis of the initial visit and followup visit are maintained
	err = doctorClient.CreateDiagnosisSet(&diaghandlers.DiagnosisListRequestData{
		VisitID: visits[0].PatientVisitID.Int64(),
		Diagnoses: []*diaghandlers.DiagnosisInputItem{
			{
				CodeID: "diag_l710",
			},
		},
	})
	test.OK(t, err)
	diagnosisListResponse, err = doctorClient.ListDiagnosis(visits[0].PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(diagnosisListResponse.Diagnoses))
	test.Equals(t, false, diagnosisListResponse.CaseManagement.Unsuitable)
	test.Equals(t, "diag_l710", diagnosisListResponse.Diagnoses[0].CodeID)

	// the diagnosis set for the initial visit should remain the same
	diagnosisListResponse, err = doctorClient.ListDiagnosis(visits[1].PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(diagnosisListResponse.Diagnoses))
	test.Equals(t, note, diagnosisListResponse.Notes)
	test.Equals(t, false, diagnosisListResponse.CaseManagement.Unsuitable)
	test.Equals(t, "diag_l710", diagnosisListResponse.Diagnoses[0].CodeID)
	test.Equals(t, "diag_l719", diagnosisListResponse.Diagnoses[1].CodeID)

}

func TestDiagnosisSet_MarkUnsuitable(t *testing.T) {
	diagnosisService := setupDiagnosisService(t)
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	// add a couple diagnosis for testing purposes
	codeID1 := "diag_l710"
	codeID2 := "diag_l719"

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	note := "testing w/ this note"
	unsuitableReason := "deal with it"
	err = doctorClient.CreateDiagnosisSet(&diaghandlers.DiagnosisListRequestData{
		VisitID:      pv.PatientVisitID,
		InternalNote: note,
		Diagnoses: []*diaghandlers.DiagnosisInputItem{
			{
				CodeID: codeID1,
			},
			{
				CodeID: codeID2,
			},
		},
		CaseManagement: diaghandlers.CaseManagementItem{
			Unsuitable: true,
			Reason:     unsuitableReason,
		},
	})
	test.OK(t, err)

	// ensure that the case is marked as being triaged out
	visit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusTriaged, visit.Status)

	diagnosisListResponse, err := doctorClient.ListDiagnosis(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, true, diagnosisListResponse.CaseManagement.Unsuitable)
	test.Equals(t, unsuitableReason, diagnosisListResponse.CaseManagement.Reason)

}
