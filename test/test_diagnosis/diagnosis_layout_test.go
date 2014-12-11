package test_diagnosis

import (
	"testing"

	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/diagnosis/icd10"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDiagnosisQuestionLayout(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// lets add a couple diagnosis codes for testing purposes
	d1 := &icd10.Diagnosis{
		Code:        "T1.0",
		Description: "Test1.0",
		Billable:    true,
	}
	d2 := &icd10.Diagnosis{
		Code:        "T2.0",
		Description: "Test2.0",
		Billable:    true,
	}
	err := icd10.SetDiagnoses(testData.DB, map[string]*icd10.Diagnosis{
		d1.Code: d1,
		d2.Code: d2,
	})
	test.OK(t, err)

	// now lets attempt to add diagnosis question info for each of the codes
	admin := test_integration.CreateRandomAdmin(t, testData)

	test_integration.UploadDetailsLayoutForDiagnosis(`
	{
	"diagnosis_layouts" : [
	{
		"code" : "T1.0",
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
		"code" : "T2.0",
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

	var codeID1, codeID2 int64
	err = testData.DB.QueryRow(`SELECT id FROM diagnosis_code WHERE code = ?`, "T1.0").Scan(&codeID1)
	test.OK(t, err)

	err = testData.DB.QueryRow(`SELECT id FROM diagnosis_code WHERE code = ?`, "T2.0").Scan(&codeID2)
	test.OK(t, err)

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
		"code" : "T1.0",
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
