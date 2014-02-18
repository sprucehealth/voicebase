package integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"carefront/apiservice"
	"carefront/common"
)

func TestAdvicePointsForPatientVisit(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	patientSignedupResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)

	// get the current primary doctor
	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	// get the patient to submit the case so that it can be reviewed by the doctor
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), patientVisitResponse.PatientVisitId, testData, t)

	// get the doctor to start reviewing the case
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// attempt to get the advice points for this patient visit
	doctorAdviceResponse := getAdvicePointsInPatientVisit(testData, doctor, patientVisitResponse.PatientVisitId, t)

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
	doctorAdviceRequest.PatientVisitId = common.NewObjectId(patientVisitResponse.PatientVisitId)

	doctorAdviceResponse = updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	if len(doctorAdviceResponse.AllAdvicePoints) != 2 {
		t.Fatal("Expected to get back the same number of advice points as were added: ")
	}

	if len(doctorAdviceResponse.SelectedAdvicePoints) != 2 {
		t.Fatal("Expected to get back the same number of advice point for patient visit as were added: ")
	}

	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// now lets go ahead and remove one point from the selection
	// note that the response now becomes the request since thats the updated view of the system
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{doctorAdviceRequest.SelectedAdvicePoints[0]}
	doctorAdviceResponse = updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)
	if len(doctorAdviceResponse.SelectedAdvicePoints) != 1 {
		t.Fatalf("Expected there to exist just 1 advice points in the selection for the patient visit. Instead there are %d", len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// now lets go ahead and update the advice for the patient visit
	doctorAdviceRequest = doctorAdviceResponse
	for i, advicePoint := range doctorAdviceRequest.AllAdvicePoints {
		advicePoint.State = common.STATE_MODIFIED
		advicePoint.Text = "UPDATED " + strconv.Itoa(i)
	}

	doctorAdviceResponse = updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets delete one of the advice points
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{doctorAdviceRequest.AllAdvicePoints[1]}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceResponse = updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse, t)

	// lets test for a bad request if an advice point that does not exist in the global list is
	// added to the patient visit
	doctorAdviceRequest.SelectedAdvicePoints = append(doctorAdviceRequest.SelectedAdvicePoints, advicePoint1)
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatal("Expected a bad request for a request that contains advice points that don't exist in the global list")
	}

	// lets start a new patient visit and ensure that we still get back the advice points as added
	patientSignedupResponse = SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse2 := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId.Int64(), testData, t)

	// get the advice points for this patient visit
	doctorAdviceResponse2 := getAdvicePointsInPatientVisit(testData, doctor, patientVisitResponse2.PatientVisitId, t)

	// there should be no selected advice points, but there should be advice points in existence
	if len(doctorAdviceResponse2.SelectedAdvicePoints) > 0 {
		t.Fatal("There should be no advice points for this particular visit given that none have been added yet")
	}

	if len(doctorAdviceResponse2.AllAdvicePoints) != 1 {
		t.Fatalf("There should exist 1 advice points given that that is what the doctor added. Instead, there exist %d", len(doctorAdviceResponse2.AllAdvicePoints))
	}

	// lets go ahead and delete all advice points
	doctorAdviceRequest = doctorAdviceResponse
	doctorAdviceRequest.AllAdvicePoints[0].State = common.STATE_DELETED
	doctorAdviceRequest.SelectedAdvicePoints = []*common.DoctorInstructionItem{}
	doctorAdviceResponse = updateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)
	if len(doctorAdviceResponse.AllAdvicePoints) > 0 {
		t.Fatal("Expected no advice points to exist given that all were deleted")
	}

	if len(doctorAdviceResponse.SelectedAdvicePoints) > 0 {
		t.Fatal("Expected no advice points to exist for patient visit given that all were deleted")
	}
}

func getAdvicePointsInPatientVisit(testData TestData, doctor *common.Doctor, patientVisitId int64, t *testing.T) *common.Advice {
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to get advice points for patient visit: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the advice points: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful call to get advice points for patient visit : "+string(body), t)

	doctorAdviceResponse := &common.Advice{}
	err = json.Unmarshal(body, doctorAdviceResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response body into the advice repsonse object: " + err.Error())
	}

	return doctorAdviceResponse
}

func updateAdvicePointsForPatientVisit(doctorAdviceRequest *common.Advice, testData TestData, doctor *common.Doctor, t *testing.T) *common.Advice {
	doctorAdviceHandler := apiservice.NewDoctorAdviceHandler(testData.DataApi)
	ts := httptest.NewServer(doctorAdviceHandler)
	defer ts.Close()

	requestBody, err := json.Marshal(doctorAdviceRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding advice points: " + err.Error())
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(requestBody), doctor.AccountId.Int64())
	if err != nil {
		t.Fatal("Unable to make successful request to add advice points to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable tp read body of the response after adding advice points to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make successful call to add advice points : "+string(body), t)

	doctorAdviceResponse := &common.Advice{}
	err = json.Unmarshal(body, doctorAdviceResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response body into json object : " + err.Error())
	}

	return doctorAdviceResponse
}

func validateAdviceRequestAgainstResponse(doctorAdviceRequest, doctorAdviceResponse *common.Advice, t *testing.T) {
	if len(doctorAdviceRequest.SelectedAdvicePoints) != len(doctorAdviceResponse.SelectedAdvicePoints) {
		t.Fatalf("Expected the same number of selected advice points in request and response. Instead request has %d while response has %d", len(doctorAdviceRequest.SelectedAdvicePoints), len(doctorAdviceResponse.SelectedAdvicePoints))
	}

	// all advice points in the global list should have ids
	for _, advicePoint := range doctorAdviceResponse.AllAdvicePoints {
		if advicePoint.Id.Int64() == 0 {
			t.Fatal("Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Advice point text is empty when not expected to be")
		}
	}

	// all advice points in the selected list should have ids
	for _, advicePoint := range doctorAdviceResponse.SelectedAdvicePoints {
		if advicePoint.Id.Int64() == 0 {
			t.Fatal("Selected Advice point expected to have an id but it doesnt")
		}
		if advicePoint.Text == "" {
			t.Fatal("Selectd advice point text is empty when not expected to be")
		}
	}

	// all updated texts should have different ids than the requests
	// all deleted advice points should not exist in the response
	// all newly added advice points should have ids
	textToIdMapping := make(map[string]int64)
	deletedAdvicePointIds := make(map[int64]bool)
	newAdvicePoints := make(map[string]bool)
	for _, advicePoint := range doctorAdviceRequest.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_MODIFIED:
			textToIdMapping[advicePoint.Text] = advicePoint.Id.Int64()

		case common.STATE_DELETED:
			deletedAdvicePointIds[advicePoint.Id.Int64()] = true

		case common.STATE_ADDED:
			newAdvicePoints[advicePoint.Text] = true
		}
	}

	for _, advicePoint := range doctorAdviceResponse.AllAdvicePoints {
		if textToIdMapping[advicePoint.Text] != 0 {
			if textToIdMapping[advicePoint.Text] == advicePoint.Id.Int64() {
				t.Fatal("Updated advice points should have different ids")
			}

			if deletedAdvicePointIds[advicePoint.Id.Int64()] == true {
				t.Fatal("Deleted advice point should not exist in the response")
			}

			if newAdvicePoints[advicePoint.Text] == true {
				if advicePoint.Id.Int64() == 0 {
					t.Fatal("Newly added advice point should have an id")
				}
			}
		}
	}

	// all
}
