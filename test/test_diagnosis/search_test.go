package test_diagnosis

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestSearchDiagnosis(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	diagnosisService := setupDiagnosisService(t)
	testData.Config.DiagnosisAPI = diagnosisService
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctorClient := test_integration.DoctorClient(testData, t, dr.DoctorID)

	// search by code that is not complete
	searchResult, err := doctorClient.SearchDiagnosis("H8")
	test.OK(t, err)
	test.Equals(t, 1, len(searchResult.Sections))
	test.Equals(t, true, len(searchResult.Sections[0].Items) > 0)
	// ensure that all subtitles start with H8
	for _, item := range searchResult.Sections[0].Items {
		test.Equals(t, true, strings.HasPrefix(item.Subtitle, "H8"))
	}

	// ensure that there is an exact code match if the right code is entered
	searchResult, err = doctorClient.SearchDiagnosis("H81.313")
	test.OK(t, err)
	test.Equals(t, 1, len(searchResult.Sections))
	test.Equals(t, 1, len(searchResult.Sections[0].Items))
	test.Equals(t, false, searchResult.Sections[0].Items[0].Diagnosis.HasDetails)

	// ensure that searching against regular word works
	searchResult, err = doctorClient.SearchDiagnosis("pregnant")
	test.OK(t, err)
	test.Equals(t, 1, len(searchResult.Sections))
	test.Equals(t, true, len(searchResult.Sections[0].Items) > 0)

	// ensure that synonym is returned for a diagnosis that has one
	searchResult, err = doctorClient.SearchDiagnosis("L70.5")
	test.OK(t, err)
	test.Equals(t, 1, len(searchResult.Sections))
	test.Equals(t, 1, len(searchResult.Sections[0].Items))
	test.Equals(t, true, searchResult.Sections[0].Items[0].Diagnosis.Synonyms != "")
	test.Equals(t, false, searchResult.Sections[0].Items[0].Diagnosis.HasDetails)

	// ensure that if we add details for the diagnosis it comes back as having details

	admin := test_integration.CreateRandomAdmin(t, testData)
	test_integration.UploadDetailsLayoutForDiagnosis(`
	{
	"diagnosis_layouts" : [
	{
		"code_id" : "diag_l705",
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

	searchResult, err = doctorClient.SearchDiagnosis("L70.5")
	test.OK(t, err)
	test.Equals(t, 1, len(searchResult.Sections))
	test.Equals(t, 1, len(searchResult.Sections[0].Items))
	test.Equals(t, true, searchResult.Sections[0].Items[0].Diagnosis.Synonyms != "")
	test.Equals(t, true, searchResult.Sections[0].Items[0].Diagnosis.HasDetails)

}
