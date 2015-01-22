package test_diagnosis

import (
	"testing"

	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDiagnosisQuestionLayout(t *testing.T) {
	diagnosisService := setupDiagnosisService(t)

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.DiagnosisAPI = diagnosisService
	testData.AdminConfig.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	// now lets attempt to add diagnosis question info for each of the codes
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
	},
	{
		"code_id" : "diag_l719",
		"layout_version" : "1.0.0",
		"questions" : [
		{
			"question" : "q_acne_severity"
		},
		{
			"question" : "q_acne_type"
		}]
	}
	]
	}`, admin.AccountID.Int64(), testData, t)

	codeID1 := "diag_l710"
	codeID2 := "diag_l719"

	// at this point there should be a diagnosis layout for each code
	// test that the layout is as expected
	modifierInfo, err := testData.DataAPI.ActiveDiagnosisDetailsIntake(codeID1, diagnosis.DetailTypes)
	test.OK(t, err)
	test.Equals(t, codeID1, modifierInfo.CodeID)
	test.Equals(t, "1.0.0", modifierInfo.Version.String())
	test.Equals(t, true, modifierInfo.Active)
	qIntake := modifierInfo.Layout.(*diagnosis.QuestionIntake)
	questions := qIntake.Questions()
	test.Equals(t, 2, len(questions))
	test.Equals(t, true, questions[0].QuestionTitle != "")
	test.Equals(t, true, questions[0].QuestionID > 0)
	test.Equals(t, "brief_title", questions[0].AdditionalFields["style"])

	modifierInfo, err = testData.DataAPI.ActiveDiagnosisDetailsIntake(codeID2, diagnosis.DetailTypes)
	test.OK(t, err)
	test.Equals(t, codeID2, modifierInfo.CodeID)
	test.Equals(t, "1.0.0", modifierInfo.Version.String())
	test.Equals(t, true, modifierInfo.Active)
	qIntake = modifierInfo.Layout.(*diagnosis.QuestionIntake)
	questions = qIntake.Questions()
	test.Equals(t, 2, len(questions))
	test.Equals(t, true, questions[0].QuestionTitle != "")
	test.Equals(t, true, questions[0].QuestionID > 0)

	// ensure that updating the layout for a code works as expected as well
	test_integration.UploadDetailsLayoutForDiagnosis(`
	{
	"diagnosis_layouts" : [
	{
		"code_id" : "diag_l710",
		"layout_version" : "1.2.0",
		"questions" : [
		{
			"question" : "q_acne_severity"
		},
		{
			"question" : "q_acne_type"
		}]
	}
	]
	}`, admin.AccountID.Int64(), testData, t)

	// ensure that the active version for T1.0 was upgraded
	modifierInfo, err = testData.DataAPI.ActiveDiagnosisDetailsIntake(codeID1, diagnosis.DetailTypes)
	test.OK(t, err)
	test.Equals(t, codeID1, modifierInfo.CodeID)
	test.Equals(t, "1.2.0", modifierInfo.Version.String())
	test.Equals(t, true, modifierInfo.Active)
}
