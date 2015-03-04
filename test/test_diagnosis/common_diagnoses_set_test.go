package test_diagnosis

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCommonDiagnosisSet(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	// create pathway
	pathway := &common.Pathway{
		Tag:            "test",
		Name:           "test",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	// create common diagnosis set
	title := "common diagnosis set"
	diagnosisCodeIDs := []string{"1", "2", "3"}

	_, err := testData.DB.Exec(`
		INSERT INTO common_diagnosis_set (title, pathway_id) VALUES (?,?)`, title, pathway.ID)
	test.OK(t, err)

	_, err = testData.DB.Exec(`
		INSERT INTO common_diagnosis_set_item (pathway_id, diagnosis_code_id)
		VALUES (?,?), (?,?), (?,?)`, pathway.ID, diagnosisCodeIDs[0], pathway.ID, diagnosisCodeIDs[1], pathway.ID, diagnosisCodeIDs[2])
	test.OK(t, err)

	// query for the common diagnosis set for the pathway
	returnedTitle, returnedDiagnosisCodeIDs, err := testData.DataAPI.CommonDiagnosisSet(pathway.Tag)
	test.OK(t, err)
	test.Equals(t, title, returnedTitle)
	test.Equals(t, diagnosisCodeIDs, returnedDiagnosisCodeIDs)

	// make one inactive and ensure that we get just the active ones back
	_, err = testData.DB.Exec(`
		UPDATE common_diagnosis_set_item 
		SET active = 0
		WHERE diagnosis_code_id = ?`, diagnosisCodeIDs[0])
	test.OK(t, err)

	returnedTitle, returnedDiagnosisCodeIDs, err = testData.DataAPI.CommonDiagnosisSet(pathway.Tag)
	test.OK(t, err)
	test.Equals(t, title, returnedTitle)
	test.Equals(t, diagnosisCodeIDs[1:], returnedDiagnosisCodeIDs)

	// ensure that if diagnosis set doesn't exist appropriate error is returned
	_, _, err = testData.DataAPI.CommonDiagnosisSet("agj")
	test.Equals(t, true, api.IsErrNotFound(err))

}
