package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestMedicationStrengthSearch(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	erx := setupErxAPI(t)
	medicationStrengthSearchHandler := &apiservice.MedicationStrengthSearchHandler{ERxApi: erx}
	ts := httptest.NewServer(medicationStrengthSearchHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?drug_internal_name="+url.QueryEscape("Benzoyl Peroxide Topical (topical - cream)"), 0)
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication strength api: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication strength api for the doctor: "+string(body), t)
	medicationStrengthResponse := &apiservice.MedicationStrengthSearchResponse{}
	err = json.Unmarshal(body, medicationStrengthResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if medicationStrengthResponse.MedicationStrengths == nil || len(medicationStrengthResponse.MedicationStrengths) == 0 {
		t.Fatal("Expected a list of medication strengths from the api but got none")
	}
}

func TestNewTreatmentSelection(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	erxApi := setupErxAPI(t)
	newTreatmentHandler := &apiservice.NewTreatmentHandler{ERxApi: erxApi}
	ts := httptest.NewServer(newTreatmentHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL+"?drug_internal_name="+url.QueryEscape("Lisinopril (oral - tablet)")+"&medication_strength="+url.QueryEscape("10 mg"), 0)
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication strength api: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication strength api for the doctor: "+string(body), t)
	newTreatmentResponse := &apiservice.NewTreatmentResponse{}
	err = json.Unmarshal(body, newTreatmentResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if newTreatmentResponse.Treatment == nil {
		t.Fatal("Expected medication object to be populated but its not")
	}

	if newTreatmentResponse.Treatment.DrugDBIds == nil || len(newTreatmentResponse.Treatment.DrugDBIds) == 0 {
		t.Fatal("Expected additional drug db ids to be returned from api but none were")
	}

	if newTreatmentResponse.Treatment.DrugDBIds[erx.LexiDrugSynId] == "0" || newTreatmentResponse.Treatment.DrugDBIds[erx.LexiSynonymTypeId] == "0" || newTreatmentResponse.Treatment.DrugDBIds[erx.LexiGenProductId] == "0" {
		t.Fatal("Expected additional drug db ids not set (lexi_drug_syn_id and lexi_synonym_type_id")
	}

	// Let's run a test for an OTC product to ensure that the OTC flag is set as expected
	resp, err = authGet(ts.URL+"?drug_internal_name="+url.QueryEscape("Fish Oil (oral - capsule)")+"&medication_strength="+url.QueryEscape("500 mg"), 0)
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication strength api: " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication strength api for the doctor for an OTC product: "+string(body), t)
	newTreatmentResponse = &apiservice.NewTreatmentResponse{}
	err = json.Unmarshal(body, newTreatmentResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if newTreatmentResponse.Treatment == nil || newTreatmentResponse.Treatment.OTC == false {
		t.Fatal("Expected the medication object to be returned and for the medication returned to be an OTC product")
	}

}

func TestDispenseUnitIds(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		return
	}

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	medicationDispenseUnitsHandler := &apiservice.MedicationDispenseUnitsHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(medicationDispenseUnitsHandler)
	defer ts.Close()

	resp, err := authGet(ts.URL, 0)
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication dispense units api: " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unable to make a successful query to the medication dispense units api for the doctor: "+string(body), t)
	medicationDispenseUnitsResponse := &apiservice.MedicationDispenseUnitsResponse{}
	err = json.Unmarshal(body, medicationDispenseUnitsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if medicationDispenseUnitsResponse.DispenseUnits == nil || len(medicationDispenseUnitsResponse.DispenseUnits) == 0 {
		t.Fatal("Expected dispense unit ids to be returned from api but none were")
	}

	for _, dispenseUnitItem := range medicationDispenseUnitsResponse.DispenseUnits {
		if dispenseUnitItem.Id == 0 || dispenseUnitItem.Text == "" {
			t.Fatal("Dispense Unit item was empty when this is not expected")
		}
	}

}

func TestAddTreatments(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
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
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId, testData, t)

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)

	// get the doctor to start reviewing the case
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{}
	treatment1.DrugInternalName = "Advil"
	treatment1.PatientVisitId = patientVisitResponse.PatientVisitId
	treatment1.DosageStrength = "10 mg"
	treatment1.DispenseValue = 1
	treatment1.DispenseUnitId = 26
	treatment1.NumberRefills = 1
	treatment1.SubstitutionsAllowed = true
	treatment1.DaysSupply = 1
	treatment1.OTC = true
	treatment1.PharmacyNotes = "testing pharmacy notes"
	treatment1.PatientInstructions = "patient instructions"
	drugDBIds := make(map[string]string)
	drugDBIds["drug_db_id_1"] = "12315"
	drugDBIds["drug_db_id_2"] = "124"
	treatment1.DrugDBIds = drugDBIds

	treatment2 := &common.Treatment{}
	treatment2.DrugInternalName = "Advil 2"
	treatment2.PatientVisitId = patientVisitResponse.PatientVisitId
	treatment2.DosageStrength = "100 mg"
	treatment2.DispenseValue = 2
	treatment2.DispenseUnitId = 27
	treatment2.NumberRefills = 3
	treatment2.SubstitutionsAllowed = false
	treatment2.DaysSupply = 12
	treatment2.OTC = false
	treatment2.PharmacyNotes = "testing pharmacy notes 2"
	treatment2.PatientInstructions = "patient instructions 2"
	drugDBIds = make(map[string]string)
	drugDBIds["drug_db_id_3"] = "12414"
	drugDBIds["drug_db_id_4"] = "214"
	treatment2.DrugDBIds = drugDBIds

	treatments := []*common.Treatment{treatment1, treatment2}

	getTreatmentsResponse := addAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	for _, treatment := range getTreatmentsResponse.Treatments {
		switch treatment.DrugInternalName {
		case treatment1.DrugInternalName:
			compareTreatments(treatment, treatment1, t)
		case treatment2.DrugInternalName:
			compareTreatments(treatment, treatment2, t)
		}
	}

	// now lets go ahead and post an update where we have just one treatment for the patient visit which was updated while the other was deleted
	treatments[0].DispenseValue = 10
	treatments = []*common.Treatment{treatments[0]}
	getTreatmentsResponse = addAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	// there should be just one treatment and its name should be the name that we just set
	if len(getTreatmentsResponse.Treatments) != 1 {
		t.Fatal("Expected just 1 treatment to be returned after update")
	}

	// the dispense value should be set to 10
	if getTreatmentsResponse.Treatments[0].DispenseValue != 10 {
		t.Fatal("Expected the updated dispense value to be set to 10")
	}

}

func TestFavoriteTreatments(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// get the current primary doctor
	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// doctor now attempts to favorite a treatment
	treatment1 := &common.Treatment{
		DrugInternalName:     "DrugName (DrugRoute - DrugForm)",
		DosageStrength:       "10 mg",
		DispenseValue:        1,
		DispenseUnitId:       26,
		NumberRefills:        1,
		SubstitutionsAllowed: true,
		DaysSupply:           1,
		OTC:                  true,
		PharmacyNotes:        "testing pharmacy notes",
		PatientInstructions:  "patient insturctions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	favoriteTreatment := &common.DoctorFavoriteTreatment{}
	favoriteTreatment.Name = "Favorite Treatment #1"
	favoriteTreatment.FavoritedTreatment = treatment1

	doctorFavoriteTreatmentsHandler := &apiservice.DoctorFavoriteTreatmentsHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorFavoriteTreatmentsHandler)
	defer ts.Close()

	favoriteTreatmentsRequest := &apiservice.DoctorFavoriteTreatmentsRequest{FavoriteTreatments: []*common.DoctorFavoriteTreatment{favoriteTreatment}}
	data, err := json.Marshal(&favoriteTreatmentsRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add favorite treatment for doctor "+string(body), t)

	favoriteTreatmentsResponse := &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if favoriteTreatmentsResponse.FavoritedTreatments == nil || len(favoriteTreatmentsResponse.FavoritedTreatments) != 1 {
		t.Fatal("Expected 1 favorited treatment in response but got none")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].Name != favoriteTreatment.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugName != "DrugName" ||
		favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugRoute != "DrugRoute" ||
		favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugForm != "DrugForm" {
		t.Fatalf("Expected the drug internal name to have been broken into its components %s %s %s", favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugName,
			favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugRoute, favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugForm)
	}

	treatment2 := &common.Treatment{
		DrugInternalName:     "DrugName2",
		DosageStrength:       "10 mg",
		DispenseValue:        1,
		DispenseUnitId:       26,
		NumberRefills:        1,
		SubstitutionsAllowed: true,
		DaysSupply:           1,
		OTC:                  true,
		PharmacyNotes:        "testing pharmacy notes",
		PatientInstructions:  "patient instructions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	favoriteTreatment2 := &common.DoctorFavoriteTreatment{}
	favoriteTreatment2.Name = "Favorite Treatment #2"
	favoriteTreatment2.FavoritedTreatment = treatment2

	favoriteTreatmentsRequest.FavoriteTreatments[0] = favoriteTreatment2
	data, err = json.Marshal(&favoriteTreatmentsRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add favorite treatment for doctor "+string(body), t)

	favoriteTreatmentsResponse = &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if favoriteTreatmentsResponse.FavoritedTreatments == nil || len(favoriteTreatmentsResponse.FavoritedTreatments) != 2 {
		t.Fatal("Expected 2 favorited treatments in response")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].Name != favoriteTreatment.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[1].Name != favoriteTreatment2.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	// lets go ahead and delete each of the treatments
	data, err = json.Marshal(&favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = authDelete(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add favorite treatment for doctor "+string(body), t)

	favoriteTreatmentsResponse = &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if len(favoriteTreatmentsResponse.FavoritedTreatments) != 0 {
		t.Fatal("Expected 1 favorited treatment after deleting the first one")
	}
}

func TestFavoriteTreatmentsInContextOfPatientVisit(t *testing.T) {
	if err := CheckIfRunningLocally(t); err == CannotRunTestLocally {
		t.Log("Skipping test since there is no database to run test on")
		return
	}
	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	// get the current primary doctor
	doctorId := getDoctorIdOfCurrentPrimaryDoctor(testData, t)

	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// create random patient
	patientSignedupResponse := SignupRandomTestPatient(t, testData.DataApi, testData.AuthApi)
	patientVisitResponse := CreatePatientVisitForPatient(patientSignedupResponse.Patient.PatientId, testData, t)
	SubmitPatientVisitForPatient(patientSignedupResponse.Patient.PatientId, patientVisitResponse.PatientVisitId, testData, t)
	StartReviewingPatientVisit(patientVisitResponse.PatientVisitId, doctor, testData, t)

	// doctor now attempts to favorite a treatment
	treatment1 := &common.Treatment{
		DrugInternalName:     "DrugName (DrugRoute - DrugForm)",
		DosageStrength:       "10 mg",
		DispenseValue:        1,
		DispenseUnitId:       26,
		NumberRefills:        1,
		SubstitutionsAllowed: true,
		DaysSupply:           1,
		OTC:                  true,
		PharmacyNotes:        "testing pharmacy notes",
		PatientInstructions:  "patient insturctions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	favoriteTreatment := &common.DoctorFavoriteTreatment{}
	favoriteTreatment.Name = "Favorite Treatment #1"
	favoriteTreatment.FavoritedTreatment = treatment1

	doctorFavoriteTreatmentsHandler := &apiservice.DoctorFavoriteTreatmentsHandler{DataApi: testData.DataApi}
	ts := httptest.NewServer(doctorFavoriteTreatmentsHandler)
	defer ts.Close()

	favoriteTreatmentsRequest := &apiservice.DoctorFavoriteTreatmentsRequest{FavoriteTreatments: []*common.DoctorFavoriteTreatment{favoriteTreatment}}
	favoriteTreatmentsRequest.PatientVisitId = patientVisitResponse.PatientVisitId
	data, err := json.Marshal(&favoriteTreatmentsRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add favorite treatment for doctor "+string(body), t)

	favoriteTreatmentsResponse := &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if favoriteTreatmentsResponse.FavoritedTreatments == nil || len(favoriteTreatmentsResponse.FavoritedTreatments) != 1 {
		t.Fatal("Expected 1 favorited treatment in response but got none")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].Name != favoriteTreatment.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugName != "DrugName" ||
		favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugRoute != "DrugRoute" ||
		favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugForm != "DrugForm" {
		t.Fatalf("Expected the drug internal name to have been broken into its components %s %s %s", favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugName,
			favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugRoute, favoriteTreatmentsResponse.FavoritedTreatments[0].FavoritedTreatment.DrugForm)
	}

	treatment2 := &common.Treatment{
		DrugInternalName:     "DrugName2 (DrugRoute - DrugForm)",
		DosageStrength:       "10 mg",
		DispenseValue:        1,
		DispenseUnitId:       26,
		NumberRefills:        1,
		SubstitutionsAllowed: true,
		DaysSupply:           1,
		OTC:                  true,
		PharmacyNotes:        "testing pharmacy notes",
		PatientInstructions:  "patient instructions",
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	// lets add this as a treatment to the patient visit
	getTreatmentsResponse := addAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment2}, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	if len(getTreatmentsResponse.Treatments) != 1 {
		t.Fatal("Expected patient visit to have 1 treatment")
	}

	// now, lets favorite a treatment that exists for the patient visit
	favoriteTreatment2 := &common.DoctorFavoriteTreatment{}
	favoriteTreatment2.Name = "Favorite Treatment #2"
	favoriteTreatment2.FavoritedTreatment = getTreatmentsResponse.Treatments[0]
	favoriteTreatmentsRequest.FavoriteTreatments[0] = favoriteTreatment2
	favoriteTreatmentsRequest.PatientVisitId = patientVisitResponse.PatientVisitId

	data, err = json.Marshal(&favoriteTreatmentsRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp2, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err = ioutil.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal("Unable to read from response body: " + err.Error())
	}
	CheckSuccessfulStatusCode(resp2, "Unsuccessful call made to add favorite treatment for doctor ", t)

	favoriteTreatmentsResponse = &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)

	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if favoriteTreatmentsResponse.FavoritedTreatments == nil || len(favoriteTreatmentsResponse.FavoritedTreatments) != 2 {
		t.Fatal("Expected 2 favorited treatments in response")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[0].Name != favoriteTreatment.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if favoriteTreatmentsResponse.FavoritedTreatments[1].Name != favoriteTreatment2.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if len(favoriteTreatmentsResponse.Treatments) == 0 {
		t.Fatal("Expected there to be 1 treatment added to the visit and the doctor")
	}

	if favoriteTreatmentsResponse.Treatments[0].DoctorFavoriteTreatmentId != favoriteTreatmentsResponse.FavoritedTreatments[1].Id {
		t.Fatal("Expected the favoriteTreatmentId to be set for the treatment and to be set to the right treatment")
	}

	// now, lets go ahead and add a treatment to the patient visit from a favorite treatment
	treatment1.DoctorFavoriteTreatmentId = favoriteTreatmentsResponse.FavoritedTreatments[0].Id
	treatment2.DoctorFavoriteTreatmentId = favoriteTreatmentsResponse.FavoritedTreatments[1].Id
	getTreatmentsResponse = addAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1, treatment2}, doctor.AccountId, patientVisitResponse.PatientVisitId, t)

	if len(getTreatmentsResponse.Treatments) != 2 {
		t.Fatal("There should exist 2 treatments for the patient visit")
	}

	if getTreatmentsResponse.Treatments[0].DoctorFavoriteTreatmentId == 0 || getTreatmentsResponse.Treatments[1].DoctorFavoriteTreatmentId == 0 {
		t.Fatal("Expected the doctorFavoriteId to be set for both treatments given that they were added from favorites")
	}

	favoriteTreatment.Id = getTreatmentsResponse.Treatments[0].DoctorFavoriteTreatmentId
	favoriteTreatment.FavoritedTreatment = getTreatmentsResponse.Treatments[0]
	favoriteTreatmentsRequest.FavoriteTreatments = []*common.DoctorFavoriteTreatment{favoriteTreatment}

	// lets delete a favorite that is also a treatment in the patient visit
	data, err = json.Marshal(&favoriteTreatmentsRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = authDelete(ts.URL, "application/json", bytes.NewBuffer(data), doctor.AccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add favorite treatment for doctor "+string(body), t)

	favoriteTreatmentsResponse = &apiservice.DoctorFavoriteTreatmentsResponse{}
	err = json.Unmarshal(body, favoriteTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if len(favoriteTreatmentsResponse.FavoritedTreatments) != 1 {
		t.Fatal("Expected 1 favorited treatment after deleting the first one")
	}

	// ensure that treatments are still returned
	if len(favoriteTreatmentsResponse.Treatments) != 2 {
		t.Fatal("Expected there to exist 2 treatments for the patient visit even after deleting one of the treatments")
	}

	if favoriteTreatmentsResponse.Treatments[0].DoctorFavoriteTreatmentId != 0 {
		t.Fatal("Expected the first treatment to no longer be a favorited treatment")
	}
}

func addAndGetTreatmentsForPatientVisit(testData TestData, treatments []*common.Treatment, doctorAccountId, PatientVisitId int64, t *testing.T) *apiservice.GetTreatmentsResponse {
	treatmentRequestBody := apiservice.AddTreatmentsRequestBody{PatientVisitId: PatientVisitId, Treatments: treatments}
	treatmentsHandler := apiservice.NewTreatmentsHandler(testData.DataApi)

	ts := httptest.NewServer(treatmentsHandler)
	defer ts.Close()

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := authPost(ts.URL, "application/json", bytes.NewBuffer(data), doctorAccountId)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add treatments for patient visit: "+string(body), t)
	addTreatmentsResponse := &apiservice.GetTreatmentsResponse{}
	err = json.Unmarshal(body, addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if addTreatmentsResponse.Treatments == nil || len(addTreatmentsResponse.Treatments) == 0 {
		t.Fatal("Treatment ids expected to be returned for the treatments just added")
	}

	return addTreatmentsResponse
}

func compareTreatments(treatment *common.Treatment, treatment1 *common.Treatment, t *testing.T) {
	if treatment.DosageStrength != treatment1.DosageStrength || treatment.DispenseValue != treatment1.DispenseValue ||
		treatment.DispenseUnitId != treatment1.DispenseUnitId || treatment.PatientInstructions != treatment1.PatientInstructions ||
		treatment.PharmacyNotes != treatment1.PharmacyNotes || treatment.NumberRefills != treatment1.NumberRefills ||
		treatment.SubstitutionsAllowed != treatment1.SubstitutionsAllowed || treatment.DaysSupply != treatment1.DaysSupply ||
		treatment.OTC != treatment1.OTC {
		treatmentData, _ := json.MarshalIndent(treatment, "", " ")
		treatment1Data, _ := json.MarshalIndent(treatment1, "", " ")

		t.Fatalf("Treatment returned from the call to get treatments for patient visit not the same as what was added for the patient visit: treatment returned: %s, treatment added: %s", string(treatmentData), string(treatment1Data))
	}

	for drugDBIdTag, drugDBId := range treatment.DrugDBIds {
		if treatment1.DrugDBIds[drugDBIdTag] == "" || treatment1.DrugDBIds[drugDBIdTag] != drugDBId {
			treatmentData, _ := json.MarshalIndent(treatment, "", " ")
			treatment1Data, _ := json.MarshalIndent(treatment1, "", " ")

			t.Fatalf("Treatment returned from the call to get treatments for patient visit not the same as what was added for the patient visit: treatment returned: %s, treatment added: %s", string(treatmentData), string(treatment1Data))
		}
	}
}
