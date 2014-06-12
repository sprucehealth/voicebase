package test_treatment_plan

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"carefront/apiservice"
	"carefront/common"
	"carefront/patient_visit"
	"carefront/test/test_integration"
)

func TestAdvicePointsForPatientVisit(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// attempt to get the advice points for this patient visit
	doctorAdviceResponse := test_integration.GetAdvicePointsInTreatmentPlan(testData, doctor, treatmentPlan.Id.Int64(), t)

	if len(doctorAdviceResponse.AllAdvicePoints) > 0 {
		t.Fatal("Expected there to be no advice points for the doctor ")
	}

	if len(doctorAdviceResponse.SelectedAdvicePoints) > 0 {
		t.Fatal("Expected there to be no advice points for the patient visit given that the doctor has not created any yet")
	}

	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	if len(doctorAdviceResponse.AllAdvicePoints) != 2 {
		t.Fatal("Expected to get back the same number of advice points as were added: ")
	} else if len(doctorAdviceResponse.SelectedAdvicePoints) != 2 {
		t.Fatal("Expected to get back the same number of advice point for patient visit as were added: ")
	} else if !doctorAdviceResponse.SelectedAdvicePoints[0].ParentId.IsValid {
		t.Fatal("Expected advice point to have a parent id but it doesnt")
	} else if !doctorAdviceResponse.SelectedAdvicePoints[1].ParentId.IsValid {
		t.Fatal("Expected advice point to have a parent id but it doesnt")
	}

	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// now lets go ahead and remove one point from the selection
	// note that the response now becomes the request since thats the updated view of the system
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{doctorAdviceRequest.SelectedAdvicePoints[0]}
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
	if len(doctorAdviceResponse.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected there to exist just 1 advice points in the selection for the patient visit. Instead there are %d", len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// now lets go ahead and update the advice for the patient visit
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	selectedAdvicePoints := make([]*common.DoctorInstructionItem, len(doctorAdviceRequest.AllAdvicePoints))
	for i, advicePoint := range doctorAdviceRequest.AllAdvicePoints {
		advicePoint.State = common.STATE_MODIFIED
		advicePoint.Text = "UPDATED " + strconv.Itoa(i)
		selectedAdvicePoints[i] = &common.DoctorInstructionItem{
			Text:     advicePoint.Text,
			ParentId: advicePoint.Id,
			State:    common.STATE_MODIFIED,
		}
	}
	doctorAdviceRequest.SelectedAdvicePoints = selectedAdvicePoints

	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets delete one of the advice points
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{doctorAdviceRequest.AllAdvicePoints[1]}
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: doctorAdviceRequest.AllAdvicePoints[0].Id,
		Text:     doctorAdviceRequest.AllAdvicePoints[0].Text,
	},
	}
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets test for the case an advice point being added to the list that does not exist in master
	doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, advicePoint1)
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := test_integration.AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	} else if resp.StatusCode != http.StatusOK {
		t.Fatal("Expected a bad request for a request that contains advice points that don't exist in the global list")
	}

	// lets start a new patient visit and ensure that we still get back the advice points as added
	_, treatmentPlan2 := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// get the advice points for this patient visit
	doctorAdviceResponse2 := test_integration.GetAdvicePointsInTreatmentPlan(testData, doctor, treatmentPlan2.Id.Int64(), t)

	// there should be no selected advice points, but there should be advice points in existence
	if len(doctorAdviceResponse2.SelectedAdvicePoints) > 0 {
		t.Fatal("There should be no advice points for this particular visit given that none have been added yet")
	}

	if len(doctorAdviceResponse2.AllAdvicePoints) != 1 {
		t.Fatalf("There should exist 1 advice points given that that is what the doctor added. Instead, there exist %d", len(doctorAdviceResponse2.AllAdvicePoints))
	}

	// lets go ahead and delete all advice points
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	doctorAdviceRequest.AllAdvicePoints = nil
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{}
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	if len(doctorAdviceResponse.AllAdvicePoints) > 0 {
		t.Fatal("Expected no advice points to exist given that all were deleted")
	}

	if len(doctorAdviceResponse.SelectedAdvicePoints) > 0 {
		t.Fatal("Expected no advice points to exist for patient visit given that all were deleted")
	}
}

func TestAdvicePointsForPatientVisit_AddAdviceOnlyToVisit(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	if len(doctorAdviceResponse.AllAdvicePoints) != 0 {
		t.Fatal("Expected to get back no advice points given none were added ")
	} else if len(doctorAdviceResponse.SelectedAdvicePoints) != 2 {
		t.Fatal("Expected to get back the same number of advice point for patient visit as were added: ")
	} else if doctorAdviceRequest.SelectedAdvicePoints[0].ParentId.IsValid {
		t.Fatal("Expected advice point to not have a parent id but it does")
	} else if doctorAdviceRequest.SelectedAdvicePoints[1].ParentId.IsValid {
		t.Fatal("Expected advice point to not have a parent id but it does")
	}

	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
}

// The purpose of this test is to ensure that we are tracking updated items
// against the original item that was added in the first place via the source_id
func TestAdvicePointsForPatientVisit_TrackingSourceId(t *testing.T) {

	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and add a couple of advice points
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// lets keep track of these two items as the source of a couple of updates
	sourceId1 := doctorAdviceResponse.AllAdvicePoints[0].Id.Int64()
	sourceId2 := doctorAdviceResponse.AllAdvicePoints[1].Id.Int64()

	// lets go ahead and modify the items
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	doctorAdviceRequest.AllAdvicePoints[0].State = common.STATE_MODIFIED
	doctorAdviceRequest.AllAdvicePoints[0].Text = "updated Advice Point 1"
	doctorAdviceRequest.AllAdvicePoints[1].State = common.STATE_MODIFIED
	doctorAdviceRequest.AllAdvicePoints[1].Text = "updated Advice Point 2"

	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// lets read the source id of the updated items and compare them
	var updatedItemSourceId1, updatedItemSourceId2 sql.NullInt64
	if err := testData.DB.QueryRow(`select source_id from dr_advice_point where id=?`, doctorAdviceResponse.AllAdvicePoints[0].Id.Int64()).Scan(&updatedItemSourceId1); err != nil {
		t.Fatalf("Attempt to get source_id for an advice point failed: %s", err)
	}

	if updatedItemSourceId1.Int64 != sourceId1 {
		t.Fatalf("Expected the sourceId of the updated item (%d) to match the id of the originating item %d ", updatedItemSourceId1.Int64, sourceId1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_advice_point where id=?`, doctorAdviceResponse.AllAdvicePoints[1].Id.Int64()).Scan(&updatedItemSourceId2); err != nil {
		t.Fatalf("Attempt to get source_id for an advice point failed: %s", err)
	}

	if updatedItemSourceId2.Int64 != sourceId2 {
		t.Fatalf("Expected the sourceId of the updated item (%d) to match the id of the originating item %d ", updatedItemSourceId2.Int64, sourceId2)
	}

	// lets go ahead and modify items once more the source id should still remain the same
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	doctorAdviceRequest.AllAdvicePoints[0].State = common.STATE_MODIFIED
	doctorAdviceRequest.AllAdvicePoints[0].Text = "updated again Advice Point 1"
	doctorAdviceRequest.AllAdvicePoints[1].State = common.STATE_MODIFIED
	doctorAdviceRequest.AllAdvicePoints[1].Text = "updated again Advice Point 2"

	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// lets read the source id of the updated items and compare them
	if err := testData.DB.QueryRow(`select source_id from dr_advice_point where id=?`, doctorAdviceResponse.AllAdvicePoints[0].Id.Int64()).Scan(&updatedItemSourceId1); err != nil {
		t.Fatalf("Attempt to get source_id for an advice point failed: %s", err)
	}

	if updatedItemSourceId1.Int64 != sourceId1 {
		t.Fatalf("Expected the sourceId of the updated item (%d) to match the id of the originating item %d ", updatedItemSourceId1.Int64, sourceId1)
	}

	if err := testData.DB.QueryRow(`select source_id from dr_advice_point where id=?`, doctorAdviceResponse.AllAdvicePoints[1].Id.Int64()).Scan(&updatedItemSourceId2); err != nil {
		t.Fatalf("Attempt to get source_id for an advice point failed: %s", err)
	}

	if updatedItemSourceId2.Int64 != sourceId2 {
		t.Fatalf("Expected the sourceId of the updated item (%d) to match the id of the originating item %d ", updatedItemSourceId2.Int64, sourceId2)
	}

}

func TestAdvicePointsForPatientVisit_AddingMultipleItemsWithSameText(t *testing.T) {

	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints = append(doctorAdviceRequest.AllAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
		doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
	}
	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
}

func TestAdvicePointsForPatientVisit_UpdatingMultipleItems(t *testing.T) {

	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints = append(doctorAdviceRequest.AllAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
		doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
	}
	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints[i].Text = "Updated text " + strconv.Itoa(i)
		doctorAdviceRequest.AllAdvicePoints[i].State = common.STATE_MODIFIED
		doctorAdviceRequest.SelectedAdvicePoints[i].Text = "Updated text " + strconv.Itoa(i)
		doctorAdviceRequest.SelectedAdvicePoints[i].State = common.STATE_MODIFIED

	}
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
}

func TestAdvicePointsForPatientVisit_SelectAdviceFromDeletedAdvice(t *testing.T) {

	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints = append(doctorAdviceRequest.AllAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
		doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
	}
	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets go ahead and delete an advice point in the context of another patient's visit

	_, treatmentPlan2 := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)
	doctorAdviceResponse2 := test_integration.GetAdvicePointsInTreatmentPlan(testData, doctor, treatmentPlan2.Id.Int64(), t)

	doctorAdviceRequest = doctorAdviceResponse2
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan2.Id
	doctorAdviceRequest.AllAdvicePoints = doctorAdviceRequest.AllAdvicePoints[:4]
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// now, lets open up the previous patient's adviceList
	doctorAdviceResponse = test_integration.GetAdvicePointsInTreatmentPlan(testData, doctor, treatmentPlan.Id.Int64(), t)

	// this should have an item in the selected advice list that does not exist in the current active list
	// Lets ensure that is true
	if len(doctorAdviceResponse.AllAdvicePoints) != 4 && len(doctorAdviceResponse.SelectedAdvicePoints) != 5 {
		t.Fatalf("Expected the global list to have 4 items and the selected list to have 5 items, instead there are %d items in the global list and %d items in the selected list", len(doctorAdviceResponse.AllAdvicePoints), len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// we should be able to submit the exact same list without having to modify anything
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan2.Id
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	if len(doctorAdviceResponse.AllAdvicePoints) != 4 && len(doctorAdviceResponse.SelectedAdvicePoints) != 5 {
		t.Fatalf("Expected the global list to have 4 items and the selected list to have 5 items, instead there are %d items in the global list and %d items in the selected list", len(doctorAdviceResponse.AllAdvicePoints), len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// now, lets go ahead and attempt to modify the last selected item in the advice list
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan2.Id
	doctorAdviceRequest.SelectedAdvicePoints[4].State = common.STATE_MODIFIED
	doctorAdviceRequest.SelectedAdvicePoints[4].Text = "Updating text of deleted item"
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// now there should still be a disparate number of items between the global and the selected list, but the last item should still be updated
	if len(doctorAdviceResponse.AllAdvicePoints) != 4 && len(doctorAdviceResponse.SelectedAdvicePoints) != 5 {
		t.Fatalf("Expected the global list to have 4 items and the selected list to have 5 items, instead there are %d items in the global list and %d items in the selected list", len(doctorAdviceResponse.AllAdvicePoints), len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// now lets go ahead and send back to the server what we got back in a response.
	// this is to ensure that the doctor can submit the advice points unmodified and pass
	// validation on the server after modifying an item in the list that is no longer in the master
	// list
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan2.Id
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	if doctorAdviceResponse.SelectedAdvicePoints[4].Text != "Updating text of deleted item" {
		t.Fatalf("Expected text to have been updated for item that is referencing a deleted item from the global list of doctor")
	}

	// now lets go ahead and remove this last item from the list
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan2.Id
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.SelectedAdvicePoints[:4]
	doctorAdviceResponse = test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
}

func TestAdvicePointsForPatientVisit_ErrorDifferentTextForLinkedItems(t *testing.T) {

	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	_, treatmentPlan, doctor := setupAdviceCreationTest(t, testData)

	// lets go ahead and create a request for this patient visit
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.SelectedAdvicePoints = make([]*common.DoctorInstructionItem, 0)
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id

	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints = append(doctorAdviceRequest.AllAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
		doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, &common.DoctorInstructionItem{
			Text:  "Advice point",
			State: common.STATE_ADDED,
		})
	}
	doctorAdviceResponse := test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	test_integration.ValidateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	doctorAdviceRequest = doctorAdviceResponse
	for i := 0; i < 5; i++ {
		doctorAdviceRequest.AllAdvicePoints[i].Text = "Updated text " + strconv.Itoa(i)
		doctorAdviceRequest.AllAdvicePoints[i].State = common.STATE_MODIFIED
		// text cannot be different for linked items
		doctorAdviceRequest.SelectedAdvicePoints[i].Text = "Updated text " + strconv.Itoa(10-i)
		doctorAdviceRequest.SelectedAdvicePoints[i].State = common.STATE_MODIFIED
	}

	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := test_integration.AuthPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a bad request for a request that contains advice points in the selected list where the text does not match the text in the global list for linked items")
	}

}

func setupAdviceCreationTest(t *testing.T, testData test_integration.TestData) (*patient_visit.PatientVisitResponse, *common.DoctorTreatmentPlan, *common.Doctor) {

	// get the current primary doctor
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse, treatmentPlan := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	return patientVisitResponse, treatmentPlan, doctor
}
