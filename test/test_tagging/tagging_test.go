package test_tagging

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestInsertTagAssociation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase2, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation("TestTag1", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	var text string
	err = testData.DB.QueryRow("SELECT tag_text FROM tag WHERE tag_text = ?", `TestTag1`).Scan(&text)
	test.OK(t, err)
	test.Equals(t, `TestTag1`, text)

	caseID2 := patientCase2.ID.Int64()
	_, err = client.InsertTagAssociation("TestTag1", &model.TagMembership{
		CaseID: &caseID2,
	})

	var tagID int64
	err = testData.DB.QueryRow("SELECT id, tag_text FROM tag WHERE tag_text = ?", `TestTag1`).Scan(&tagID, &text)
	test.OK(t, err)
	test.Equals(t, `TestTag1`, text)

	rows, err := testData.DB.Query("SELECT case_id FROM tag_membership WHERE tag_id = ?", tagID)
	test.OK(t, err)

	var found int
	expected := []int64{caseID, caseID2}
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
		test.OK(t, err)
		test.Equals(t, expected[found], id)
		found++
	}
	test.OK(t, rows.Err())
	test.Equals(t, 2, found)
}

func TestGetTags(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation("TestTag1", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("TestTag2", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("TestTag3", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("FooTag2", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	found := make(map[string]bool)
	tags, err := client.Tags([]string{"Test"})
	test.OK(t, err)
	test.Equals(t, len(tags), 3)
	fooTags, err := client.Tags([]string{"Foo"})
	test.OK(t, err)
	test.Equals(t, len(fooTags), 1)
	for _, v := range tags {
		found[v.Text] = true
	}
	for _, v := range fooTags {
		found[v.Text] = true
	}
	_, ok := found["TestTag1"]
	test.Assert(t, ok, "Did not find expected tag")
	_, ok = found["TestTag2"]
	test.Assert(t, ok, "Did not find expected tag")
	_, ok = found["TestTag3"]
	test.Assert(t, ok, "Did not find expected tag")
	_, ok = found["FooTag2"]
	test.Assert(t, ok, "Did not find expected tag")
}

func TestDeleteTag(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation("TestTag1", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("TestTag2", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.Tags([]string{"Test"})
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
	for _, v := range tags {
		aff, err := client.DeleteTag(v.ID)
		test.OK(t, err)
		test.Equals(t, int64(1), aff)
	}
	tags, err = client.Tags([]string{"Test"})
	test.OK(t, err)
	test.Equals(t, 0, len(tags))
}

func TestTagAssociations(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase2, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase3, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	caseID2 := patientCase2.ID.Int64()
	caseID3 := patientCase3.ID.Int64()
	_, err := client.InsertTagAssociation("A", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("B", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("C", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("A", &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("B", &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("A", &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("D", &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("E", &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("F", &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)

	ms, err := client.TagMembershipQuery(`A`)
	test.OK(t, err)
	associations, err := client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`B`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`C`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 1, len(associations))

	ms, err = client.TagMembershipQuery(`A | B | D`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`!D`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`A AND (E OR F)`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`A AND (E OR F AND (NOT D))`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 1, len(associations))

	ms, err = client.TagMembershipQuery(`A OR (E OR F AND (NOT D))`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`!A`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 0, len(associations))

	ms, err = client.TagMembershipQuery(`NotValid`)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms)
	test.OK(t, err)
	test.Equals(t, 0, len(associations))
}

func TestDeleteTagCaseAssociation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation("TestTag1", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation("TestTag2", &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.Tags([]string{"Test"})
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
	err = client.DeleteTagCaseAssociation("TestTag2", caseID)
	test.OK(t, err)
	tagAs, err := client.TagMembershipQuery("TestTag2")
	test.OK(t, err)
	test.Equals(t, 0, len(tagAs))
	err = client.DeleteTagCaseAssociation("TestTag1", caseID)
	test.OK(t, err)
	tagAs, err = client.TagMembershipQuery("TestTag1")
	test.OK(t, err)
	test.Equals(t, 0, len(tagAs))
	tags, err = client.Tags([]string{"Test"})
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
}

func createPatientCaseAndAssignToDoctor(t *testing.T, testData *test_integration.TestData) (*common.PatientCase, *common.Doctor) {
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	// Create a random patient
	vp, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// Create a visit/case for the patient visit
	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(vp.PatientVisitID)
	test.OK(t, err)

	return patientCase, doctor
}
