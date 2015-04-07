package test_treatment_plan

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestRegimenForPatientVisit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// attempt to get the regimen plan or a patient visit
	regimenPlan := test_integration.GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan.ID.Int64(), t)

	if len(regimenPlan.AllSteps) > 0 {
		t.Fatal("There should be no regimen steps given that none have been created yet")
	}

	if len(regimenPlan.Sections) > 0 {
		t.Fatal("There should be no regimen sections for the patient visit given that none have been created yet")
	}

	// adding new regimen steps to the doctor but not to the patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.StateAdded,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.StateAdded,
	}

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.Sections) > 0 {
		t.Fatal("Regimen section should not exist even though regimen steps were created by doctor")
	}

	// make the response the request since the response always returns the updated view of the system
	regimenPlanRequest = regimenPlanResponse

	// now lets add a couple regimen steps to a regimen section
	regimenSection := &common.RegimenSection{}
	regimenSection.Name = "morning"
	regimenSection.Steps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentID: regimenPlanRequest.AllSteps[0].ID,
		Text:     regimenPlanRequest.AllSteps[0].Text,
	},
	}

	regimenSection2 := &common.RegimenSection{}
	regimenSection2.Name = "night"
	regimenSection2.Steps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentID: regimenPlanRequest.AllSteps[1].ID,
		Text:     regimenPlanRequest.AllSteps[1].Text,
	},
	}

	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanResponse, regimenPlanResponse, t)

	if len(regimenPlanResponse.Sections) != 2 {
		t.Fatalf("Expected the number of regimen sections to be 2 but there are %d instead", len(regimenPlanResponse.Sections))
	} else if !regimenPlanResponse.Sections[0].Steps[0].ParentID.IsValid {
		t.Fatalf("Expected the regimen step to have a parent id but it doesnt")
	} else if !regimenPlanResponse.Sections[0].Steps[0].ParentID.IsValid {
		t.Fatalf("Expected the regimen step to have a parent id but it doesnt")
	}

	// now remove a section from the request
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.Sections = []*common.RegimenSection{regimenPlanRequest.Sections[0]}

	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.Sections) != 1 {
		t.Fatalf("Expected the number of regimen sections to be 2 but there are %d instead", len(regimenPlanResponse.Sections))
	}

	// lets update a regimen step in the section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllSteps[0].Text = "UPDATED 1"
	regimenPlanRequest.AllSteps[0].State = common.StateModified
	regimenPlanRequest.Sections[0].Steps[0].Text = "UPDATED 1"
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// lets delete a regimen step
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenPlanRequest.AllSteps[0]}
	regimenPlanRequest.Sections = []*common.RegimenSection{}
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
	if len(regimenPlanResponse.AllSteps) != 1 {
		t.Fatal("Should only have 1 regimen step given that we just deleted one from the list")
	}

	// lets attempt to remove the regimen step, but keep it in the regimen section.
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{}
	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection}

	requestBody, err := json.Marshal(regimenPlanRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding regimen steps: " + err.Error())
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorRegimenURLPath, "application/json", bytes.NewBuffer(requestBody), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to create regimen for patient visit")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatal("Expected to get a bad request for when the regimen step does not exist in the regimen sections")
	}

	// get patient to start a visit

	_, treatmentPlan = test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	regimenPlan = test_integration.GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan.ID.Int64(), t)
	if len(regimenPlan.Sections) > 0 {
		t.Fatal("There should not be any regimen sections for a new patient visit")
	}

	if len(regimenPlan.AllSteps) != 0 {
		t.Fatal("There should be no regimen steps existing globally for this doctor")
	}
}

func TestRegimenForPatientVisit_AddOnlyToPatientVisit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// add regimen steps only to section and not to master list
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.StateAdded,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.StateAdded,
	}

	regimenSection := &common.RegimenSection{
		Name: "morning",
		Steps: []*common.DoctorInstructionItem{{
			Text: regimenStep1.Text,
		}},
	}

	regimenSection2 := &common.RegimenSection{
		Name: "night",
		Steps: []*common.DoctorInstructionItem{{
			Text: regimenStep2.Text,
		}},
	}

	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.Sections) != 2 {
		t.Fatalf("Expected 2 regimen sections but got %d", len(regimenPlanResponse.Sections))
	} else if regimenPlanRequest.Sections[0].Steps[0].ParentID.IsValid {
		t.Fatal("Expected parent id to not exist for regimen step but it does")
	} else if regimenPlanRequest.Sections[1].Steps[0].ParentID.IsValid {
		t.Fatal("Expected parent id to not exist for regimen step but it does")
	}

}

func TestRegimenForPatientVisit_AddingMultipleItemsWithSameText(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
		AllSteps:        make([]*common.DoctorInstructionItem, 0),
	}

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps = append(regimenPlanRequest.AllSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.StateAdded,
		})

		regimenPlanRequest.Sections = append(regimenPlanRequest.Sections, &common.RegimenSection{
			Name: "test " + strconv.Itoa(i),
			Steps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.StateAdded,
			},
			},
		})
	}

	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

}

// The purpose of this test is to ensure that if the client specified text in the regimen section
// that does not match up with the text in the master regimen list when the linkage between the two exists,
// we accept what the client gives us as being present in the regimen section and the master regimen list
// but break the linkage given that the text differs
func TestRegimenForPatientVisit_TextDifferentForLinkedItem(t *testing.T) {

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
		AllSteps:        make([]*common.DoctorInstructionItem, 0),
	}

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps = append(regimenPlanRequest.AllSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.StateAdded,
		})

		regimenPlanRequest.Sections = append(regimenPlanRequest.Sections, &common.RegimenSection{
			Name: "test " + strconv.Itoa(i),
			Steps: []*common.DoctorInstructionItem{{
				Text:  "Regimen Step",
				State: common.StateAdded,
			}},
		})
	}

	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// all steps in the response should have a parent id
	for i := 0; i < 5; i++ {
		parentID := regimenPlanResponse.Sections[i].Steps[0].ParentID
		if !parentID.IsValid || parentID.Int64() == 0 {
			t.Fatalf("Expected parentId to exist")
		}
	}

	regimenPlanRequest = regimenPlanResponse

	// lets go ahead and update each item in the list
	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps[i].Text = "Updated Regimen Step"
		regimenPlanRequest.AllSteps[i].State = common.StateModified

		regimenPlanRequest.Sections[i].Steps[0].Text = "Updated Regimen Step " + strconv.Itoa(i)
		regimenPlanRequest.Sections[i].Steps[0].State = common.StateModified
	}

	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// ensure that none of the steps in the regimen sections have a parent id
	for _, regimenSection := range regimenPlanResponse.Sections {
		test.Equals(t, false, regimenSection.Steps[0].ParentID.IsValid)
	}
}

func TestRegimenForPatientVisit_UpdatingMultipleItemsWithSameText(t *testing.T) {

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
		AllSteps:        make([]*common.DoctorInstructionItem, 0),
	}

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps = append(regimenPlanRequest.AllSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.StateAdded,
		})

		regimenPlanRequest.Sections = append(regimenPlanRequest.Sections, &common.RegimenSection{
			Name: "test " + strconv.Itoa(i),
			Steps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.StateAdded,
			},
			},
		})
	}

	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	regimenPlanRequest = regimenPlanResponse

	// lets go ahead and update each item in the list
	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps[i].Text = "Updated Regimen Step"
		regimenPlanRequest.AllSteps[i].State = common.StateModified

		regimenPlanRequest.Sections[i].Steps[0].Text = "Updated Regimen Step"
		regimenPlanRequest.Sections[i].Steps[0].State = common.StateModified
	}

	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
}

func TestRegimenForPatientVisit_UpdatingItemLinkedToDeletedItem(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// add multiple items with the exact same text and ensure that they all get assigned new ids
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanID = treatmentPlan.ID
	regimenPlanRequest.AllSteps = make([]*common.DoctorInstructionItem, 0)

	for i := 0; i < 5; i++ {
		regimenPlanRequest.AllSteps = append(regimenPlanRequest.AllSteps, &common.DoctorInstructionItem{
			Text:  "Regimen Step",
			State: common.StateAdded,
		})

		regimenPlanRequest.Sections = append(regimenPlanRequest.Sections, &common.RegimenSection{
			Name: "test " + strconv.Itoa(i),
			Steps: []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
				Text:  "Regimen Step",
				State: common.StateAdded,
			},
			},
		})
	}

	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// now lets update the global set of regimen steps in the context of another patient's visit
	_, treatmentPlan2 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	regimenPlanResponse = test_integration.GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan2.ID.Int64(), t)

	// lets go ahead and delete one of the items from the regimen step
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.TreatmentPlanID = treatmentPlan2.ID
	regimenPlanRequest.AllSteps = regimenPlanRequest.AllSteps[0:4]

	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	if len(regimenPlanResponse.AllSteps) != 4 {
		t.Fatalf("Expected there to exist 4 items in the global regimen steps after deleting one of them instead got %d items ", len(regimenPlanResponse.AllSteps))
	}

	// now, lets go back to the previous patient and attempt to get the regimen plan
	regimenPlanResponse = test_integration.GetRegimenPlanForTreatmentPlan(testData, doctor, treatmentPlan.ID.Int64(), t)
	if len(regimenPlanResponse.AllSteps) != 4 && len(regimenPlanResponse.Sections) != 5 {
		t.Fatalf("Expected 4 items in the global regimen steps and 5 items in the regimen sections instead got %d in global regimen list and %d items in the regimen sections", len(regimenPlanRequest.AllSteps), len(regimenPlanRequest.Sections))
	}

	// now lets go ahead and try and modify the item in the regimen section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.Sections[4].Steps[0].State = common.StateModified
	regimenPlanRequest.TreatmentPlanID = treatmentPlan2.ID
	updatedText := "Updating text for an item linked to deleted item"
	regimenPlanRequest.Sections[4].Steps[0].Text = updatedText

	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	if len(regimenPlanResponse.AllSteps) != 4 && len(regimenPlanResponse.Sections) != 5 {
		t.Fatalf("Expected 4 items in the global regimen steps and 5 items in the regimen sections instead got %d in global regimen list and %d items in the regimen sections", len(regimenPlanRequest.AllSteps), len(regimenPlanRequest.Sections))
	}

	if regimenPlanResponse.Sections[4].Steps[0].Text != updatedText {
		t.Fatalf("Exepcted text to have updated for item linked to deleted item but it didn't")
	}

	// now lets go ahead and echo back the response to the server to ensure that it takes the list
	// as it modified back without any issue. This is essentially to ensure that it passes the validation
	// of text being modified for an item that is no longer active in the master list
	regimenPlanRequest = regimenPlanResponse
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// now lets go ahead and remove the item from the regimen section
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.Sections = regimenPlanRequest.Sections[:4]
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)
}

// The purpose of this test is to ensure that when regimen steps are updated,
// we are keeping track of the original step that has been modified via a source_id
func TestRegimenForPatientVisit_TrackingSourceId(t *testing.T) {

	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	_, treatmentPlan, doctor := setupTestForRegimenCreation(t, testData)

	// adding new regimen steps to the doctor but not to the patient visit
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanID = treatmentPlan.ID

	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.StateAdded

	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.StateAdded

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse := test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	if len(regimenPlanResponse.Sections) > 0 {
		t.Fatal("Regimen section should not exist even though regimen steps were created by doctor")
	}

	// keep track of the source ids of both steps
	sourceID1 := regimenPlanResponse.AllSteps[0].ID.Int64()
	sourceID2 := regimenPlanResponse.AllSteps[1].ID.Int64()

	// lets update both steps
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllSteps[0].State = common.StateModified
	regimenPlanRequest.AllSteps[0].Text = "Updated step 1"
	regimenPlanRequest.AllSteps[1].State = common.StateModified
	regimenPlanRequest.AllSteps[1].Text = "Updated step 2"
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// the source id of the two returned steps should match the source id of the original steps
	var updatedItemSourceID1, updatedItemSourceID2 sql.NullInt64
	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllSteps[0].ID.Int64()).Scan(&updatedItemSourceID1); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceID1.Int64 != sourceID1 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceID1.Int64, sourceID1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllSteps[1].ID.Int64()).Scan(&updatedItemSourceID2); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceID2.Int64 != sourceID2 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceID2.Int64, sourceID2)
	}

	// lets update again and the source id should still match
	regimenPlanRequest = regimenPlanResponse
	regimenPlanRequest.AllSteps[0].State = common.StateModified
	regimenPlanRequest.AllSteps[0].Text = "Updated again step 1"
	regimenPlanRequest.AllSteps[1].State = common.StateModified
	regimenPlanRequest.AllSteps[1].Text = "Updated again step 2"
	regimenPlanResponse = test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)
	test_integration.ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// the source id of the two returned steps should match the source id of the original steps
	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllSteps[0].ID.Int64()).Scan(&updatedItemSourceID1); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceID1.Int64 != sourceID1 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceID1.Int64, sourceID1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_regimen_step where id=?`, regimenPlanResponse.AllSteps[1].ID.Int64()).Scan(&updatedItemSourceID2); err != nil {
		t.Fatalf("Expected the query to get source_id to succeed instead it failed: %s", err)
	}

	if updatedItemSourceID2.Int64 != sourceID2 {
		t.Fatalf("Expected the sourceId retrieved from the updated item (%d) to match the id of the original item (%d)", updatedItemSourceID2.Int64, sourceID2)
	}

}

func setupTestForRegimenCreation(t *testing.T, testData *test_integration.TestData) (*patient.PatientVisitResponse, *common.TreatmentPlan, *common.Doctor) {
	// get the current primary doctor
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	return patientVisitResponse, treatmentPlan, doctor
}
