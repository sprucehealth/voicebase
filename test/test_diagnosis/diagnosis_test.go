package test_diagnosis

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestGetDiagnosis(t *testing.T) {
	diagnosisService := setupDiagnosisService(t)
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.Config.DiagnosisAPI = diagnosisService
	testData.AdminConfig.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	codeID1 := "diag_l710"
	codeID2 := "diag_l719"

	diagnosisMap, err := diagnosisService.DiagnosisForCodeIDs([]string{codeID1, codeID2})
	test.OK(t, err)
	d1 := diagnosisMap[codeID1]
	d2 := diagnosisMap[codeID2]

	// add layout for diag_l710
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

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	// attempt to get each diagnosis item and ensure that output is as expected
	d1OutputItem, err := doctorClient.GetDiagnosis(codeID1)
	test.OK(t, err)
	test.Equals(t, codeID1, d1OutputItem.CodeID)
	test.Equals(t, d1.Code, d1OutputItem.Code)
	test.Equals(t, d1.Description, d1OutputItem.Title)
	test.Equals(t, true, d1OutputItem.HasDetails)
	test.Equals(t, 2, len(d1OutputItem.Questions))
	test.Equals(t, "1.0.0", d1OutputItem.LatestLayoutVersion.String())
	test.Equals(t, "1.0.0", d1OutputItem.LayoutVersion.String())

	d2OutputItem, err := doctorClient.GetDiagnosis(codeID2)
	test.OK(t, err)
	test.Equals(t, codeID2, d2OutputItem.CodeID)
	test.Equals(t, d2.Code, d2OutputItem.Code)
	test.Equals(t, d2.Description, d2OutputItem.Title)
	test.Equals(t, false, d2OutputItem.HasDetails)
}
