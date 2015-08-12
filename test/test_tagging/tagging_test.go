package test_tagging

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/tagging"
	"github.com/sprucehealth/backend/tagging/model"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestInsertTagAssociation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase2, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	var text string
	err = testData.DB.QueryRow("SELECT tag_text FROM tag WHERE tag_text = ?", `TestTag1`).Scan(&text)
	test.OK(t, err)
	test.Equals(t, `TestTag1`, text)

	caseID2 := patientCase2.ID.Int64()
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
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
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag3"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "FooTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	found := make(map[string]bool)
	tags, err := client.TagsFromText([]string{"Test"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, len(tags), 3)
	fooTags, err := client.TagsFromText([]string{"Foo"}, tagging.TONone)
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
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.TagsFromText([]string{"Test"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
	for _, v := range tags {
		aff, err := client.DeleteTag(v.ID)
		test.OK(t, err)
		test.Equals(t, int64(1), aff)
	}
	tags, err = client.TagsFromText([]string{"Test"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 0, len(tags))
}

func TestTagAssociations(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase2, _ := createPatientCaseAndAssignToDoctor(t, testData)
	patientCase3, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	caseID2 := patientCase2.ID.Int64()
	caseID3 := patientCase3.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "A"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "B"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "C"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "A"}, &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "B"}, &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "A"}, &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "D"}, &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "E"}, &model.TagMembership{
		CaseID: &caseID2,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "F"}, &model.TagMembership{
		CaseID: &caseID3,
	})
	test.OK(t, err)

	ms, err := client.TagMembershipQuery(`A`, tagging.TONone)
	test.OK(t, err)
	associations, err := client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`B`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`C`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 1, len(associations))

	ms, err = client.TagMembershipQuery(`A | B | D`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`!D`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`A AND (E OR F)`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 2, len(associations))

	ms, err = client.TagMembershipQuery(`A AND (E OR F AND (NOT D))`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 1, len(associations))

	ms, err = client.TagMembershipQuery(`A OR (E OR F AND (NOT D))`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 3, len(associations))

	ms, err = client.TagMembershipQuery(`!A`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 0, len(associations))

	ms, err = client.TagMembershipQuery(`NotValid`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Unix(1, 0).Unix(), time.Now().Unix()+60)
	test.OK(t, err)
	test.Equals(t, 0, len(associations))

	// Test that we can timebound our query by visit
	ms, err = client.TagMembershipQuery(`A`, tagging.TONone)
	test.OK(t, err)
	associations, err = client.CaseAssociations(ms, time.Now().Unix()+60, time.Now().Unix()+62)
	test.OK(t, err)
	test.Equals(t, 0, len(associations))
}

func TestDeleteTagCaseAssociation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.TagsFromText([]string{"Test"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
	err = client.DeleteTagCaseAssociation("TestTag2", caseID)
	test.OK(t, err)
	tagAs, err := client.TagMembershipQuery("TestTag2", tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 0, len(tagAs))
	err = client.DeleteTagCaseAssociation("TestTag1", caseID)
	test.OK(t, err)
	tagAs, err = client.TagMembershipQuery("TestTag1", tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 0, len(tagAs))
	tags, err = client.TagsFromText([]string{"Test"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 2, len(tags))
}

func TestTagsMapping(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.TagsFromText([]string{"TestTag1", "TestTag2"}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 2, len(tags))

	tagMap, err := client.Tags([]int64{tags[0].ID, tags[1].ID})
	test.OK(t, err)
	test.Equals(t, 2, len(tagMap))
	test.Equals(t, tags[0], tagMap[tags[0].ID])
	test.Equals(t, tags[1], tagMap[tags[1].ID])
}

func TestTagsForCases(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)

	client := tagging.NewTaggingClient(testData.DB)
	caseID := patientCase.ID.Int64()
	_, err := client.InsertTagAssociation(&model.Tag{Text: "TestTag1"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)
	_, err = client.InsertTagAssociation(&model.Tag{Text: "TestTag2"}, &model.TagMembership{
		CaseID: &caseID,
	})
	test.OK(t, err)

	tags, err := client.TagsForCases([]int64{caseID}, tagging.TONone)
	test.OK(t, err)
	test.Equals(t, 1, len(tags))
	test.Equals(t, 12, len(tags[caseID]))

	tags, err = client.TagsForCases([]int64{caseID}, tagging.TONonHiddenOnly)
	test.OK(t, err)
	test.Equals(t, 1, len(tags))
	test.Equals(t, 6, len(tags[caseID]))
}

func TestCCTagRoundTrip(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	taggingCli := tagging.NewTaggingClient(testData.DB)
	cc, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	doctorCli := test_integration.DoctorClient(testData, t, cc.DoctorID)
	taggingCli.InsertTag(&model.Tag{Text: "TestTag1", Common: false})
	taggingCli.InsertTag(&model.Tag{Text: "TestTag2", Common: true})
	taggingCli.InsertTag(&model.Tag{Text: "OtherTag1", Common: false})

	getResp, err := doctorCli.Tags(&tagging.TagGETRequest{Text: []string{}, Common: true})
	test.OK(t, err)
	test.Assert(t, len(getResp.Tags) == 1, "Expected 1 common tag to be returned but got %v", getResp.Tags)
	test.Equals(t, "TestTag2", getResp.Tags[0].Text)

	// The server should be sorting so we can assert alpha order DESC and  prefix search
	getResp, err = doctorCli.Tags(&tagging.TagGETRequest{Text: []string{"T"}})
	test.OK(t, err)
	test.Assert(t, len(getResp.Tags) == 2, "Expected 2 tags tag to be returned but got %v", getResp.Tags)
	test.Equals(t, "TestTag2", getResp.Tags[0].Text)
	test.Equals(t, "TestTag1", getResp.Tags[1].Text)

	getResp, err = doctorCli.Tags(&tagging.TagGETRequest{Text: []string{"O"}})
	test.OK(t, err)
	test.Assert(t, len(getResp.Tags) == 1, "Expected 1 tag tag to be returned but got %v", getResp.Tags)
	test.Equals(t, "OtherTag1", getResp.Tags[0].Text)

	test.OK(t, doctorCli.DeleteTag(&tagging.TagDELETERequest{ID: getResp.Tags[0].ID}))
	getResp, err = doctorCli.Tags(&tagging.TagGETRequest{Text: []string{"O"}})
	test.OK(t, err)
	test.Assert(t, len(getResp.Tags) == 0, "Expected 0 tags tag to be returned but got %v", getResp.Tags)
}

func TestCCTagCaseMembershipAndAssociationRoundTrip(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	cc, _, _ := test_integration.SignupRandomTestCC(t, testData, true)
	doctorCli := test_integration.DoctorClient(testData, t, cc.DoctorID)

	patientCase, _ := createPatientCaseAndAssignToDoctor(t, testData)
	caseID := patientCase.ID.Int64()
	postTagCaseAssociationResp, err := doctorCli.PostTagCaseAssociation(&tagging.TagCaseAssociationPOSTRequest{
		Text:   "TestTag1",
		CaseID: &caseID,
	})
	test.OK(t, err)
	test.Assert(t, postTagCaseAssociationResp.TagID != 0, "Expected a non zero tag ID to e returned but got %v", postTagCaseAssociationResp)
	getTagCaseMembershipResp, err := doctorCli.TagCaseMemberships(&tagging.TagCaseMembershipGETRequest{CaseID: caseID})
	test.OK(t, err)
	var found bool
	for _, v := range getTagCaseMembershipResp.TagMemberships {
		if v.TagID == postTagCaseAssociationResp.TagID {
			found = true
		}
	}
	test.Assert(t, found, "Did not find previously created membership in GET response")

	postTagCaseAssociationResp, err = doctorCli.PostTagCaseAssociation(&tagging.TagCaseAssociationPOSTRequest{
		Text:   "TestTag2",
		CaseID: &caseID,
	})
	test.OK(t, err)
	triggerTime := time.Now().Unix() - 5
	test.OK(t, doctorCli.PutTagCaseMembership(&tagging.TagCaseMembershipPUTRequest{
		CaseID:      caseID,
		TagID:       postTagCaseAssociationResp.TagID,
		TriggerTime: &triggerTime,
	}))

	getTagCaseAssociationResp, err := doctorCli.TagCaseAssociations(&tagging.TagCaseAssociationGETRequest{
		Query:       "",
		Start:       time.Unix(1, 0).Unix(),
		End:         time.Now().Unix() + 60,
		PastTrigger: true,
	})
	test.OK(t, err)
	test.Equals(t, 1, len(getTagCaseAssociationResp.Associations))
	test.Equals(t, caseID, getTagCaseAssociationResp.Associations[0].ID)

	postTagCaseAssociationResp, err = doctorCli.PostTagCaseAssociation(&tagging.TagCaseAssociationPOSTRequest{
		Text:   "TestTag3",
		CaseID: &caseID,
	})
	test.OK(t, err)
	test.OK(t, doctorCli.DeleteTagCaseMembership(&tagging.TagCaseMembershipDELETERequest{
		CaseID: caseID,
		TagID:  postTagCaseAssociationResp.TagID,
	}))
	getTagCaseMembershipResp, err = doctorCli.TagCaseMemberships(&tagging.TagCaseMembershipGETRequest{CaseID: caseID})
	test.OK(t, err)
	notFound := true
	for _, v := range getTagCaseMembershipResp.TagMemberships {
		if v.TagID == postTagCaseAssociationResp.TagID {
			notFound = false
		}
	}
	test.Assert(t, notFound, "Expected to not find previously created membership in GET response")

	postTagCaseAssociationResp, err = doctorCli.PostTagCaseAssociation(&tagging.TagCaseAssociationPOSTRequest{
		Text:   "TestTag4",
		CaseID: &caseID,
	})
	test.OK(t, err)
	test.OK(t, doctorCli.DeleteTagCaseAssociation(&tagging.TagCaseAssociationDELETERequest{
		Text:   "TestTag4",
		CaseID: caseID,
	}))
	notFound = true
	for _, v := range getTagCaseMembershipResp.TagMemberships {
		if v.TagID == postTagCaseAssociationResp.TagID {
			notFound = false
		}
	}
	test.Assert(t, notFound, "Expected to not find previously created membership in GET response")

	_, err = doctorCli.PostTagCaseAssociation(&tagging.TagCaseAssociationPOSTRequest{
		Text:   "TestTag5",
		CaseID: &caseID,
	})
	test.OK(t, err)
	getResp, err := doctorCli.Tags(&tagging.TagGETRequest{Text: []string{"TestTag5"}, Common: false})
	test.OK(t, err)
	test.Assert(t, len(getResp.Tags) == 1, "Expected 1 tag to be returned but got %v", getResp.Tags)
	test.Equals(t, "TestTag5", getResp.Tags[0].Text)
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
