package test_diagnosis

import (
	"testing"

	"github.com/sprucehealth/backend/diagnosis/icd10"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestGetDiagnosis(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// add a couple diagnosis for testing purposes
	code1 := "T1.0"
	d1 := &icd10.Diagnosis{
		Code:        code1,
		Description: "Test1.0",
		Billable:    true,
	}

	code2 := "T2.0"
	d2 := &icd10.Diagnosis{
		Code:        code2,
		Description: "Test2.0",
		Billable:    true,
	}
	err := icd10.SetDiagnoses(testData.DB, map[string]*icd10.Diagnosis{
		d1.Code: d1,
		d2.Code: d2,
	})
	test.OK(t, err)

	// add layout for T1.0
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
	}
	]
	}`, admin.AccountID.Int64(), testData, t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	var codeID1, codeID2 int64
	err = testData.DB.QueryRow(`SELECT id FROM diagnosis_code WHERE code = ?`, code1).Scan(&codeID1)
	test.OK(t, err)
	err = testData.DB.QueryRow(`SELECT id FROM diagnosis_code WHERE code = ?`, code2).Scan(&codeID2)
	test.OK(t, err)

	// attempt to get each diagnosis item and ensure that output is as expected
	d1OutputItem, err := doctorClient.GetDiagnosis(codeID1)
	test.OK(t, err)
	test.Equals(t, codeID1, d1OutputItem.CodeID)
	test.Equals(t, code1, d1OutputItem.Code)
	test.Equals(t, d1.Description, d1OutputItem.Title)
	test.Equals(t, true, d1OutputItem.HasDetails)
	test.Equals(t, 2, len(d1OutputItem.Questions))
	test.Equals(t, "1.0.0", d1OutputItem.LatestLayoutVersion.String())
	test.Equals(t, "1.0.0", d1OutputItem.LayoutVersion.String())

	d2OutputItem, err := doctorClient.GetDiagnosis(codeID2)
	test.OK(t, err)
	test.Equals(t, codeID2, d2OutputItem.CodeID)
	test.Equals(t, code2, d2OutputItem.Code)
	test.Equals(t, d2.Description, d2OutputItem.Title)
	test.Equals(t, false, d2OutputItem.HasDetails)
}
