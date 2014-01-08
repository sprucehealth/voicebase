package integration

import (
	"bytes"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/erx"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

	resp, err := http.Get(ts.URL + "?drug_internal_name=" + url.QueryEscape("Benzoyl Peroxide Topical (topical - cream)"))
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

	resp, err := http.Get(ts.URL + "?drug_internal_name=" + url.QueryEscape("Lisinopril (oral - tablet)") + "&medication_strength=" + url.QueryEscape("10 mg"))
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
	resp, err = http.Get(ts.URL + "?drug_internal_name=" + url.QueryEscape("Fish Oil (oral - capsule)") + "&medication_strength=" + url.QueryEscape("500 mg"))
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

	resp, err := http.Get(ts.URL)
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
	patientVisitResponse := GetPatientVisitForPatient(patientSignedupResponse.PatientId, testData, t)

	// get patient to submit the visit
	SubmitPatientVisitForPatient(patientSignedupResponse.PatientId, patientVisitResponse.PatientVisitId, testData, t)

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

func addAndGetTreatmentsForPatientVisit(testData TestData, treatments []*common.Treatment, DoctorAccountId, PatientVisitId int64, t *testing.T) *apiservice.GetTreatmentsResponse {
	treatmentRequestBody := apiservice.AddTreatmentsRequestBody{PatientVisitId: PatientVisitId, Treatments: treatments}
	treatmentsHandler := apiservice.NewTreatmentsHandler(testData.DataApi)
	treatmentsHandler.AccountIdFromAuthToken(DoctorAccountId)

	ts := httptest.NewServer(treatmentsHandler)

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := http.Post(ts.URL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to add treatments for patient visit: "+string(body), t)
	addTreatmentsResponse := &apiservice.AddTreatmentsResponse{}
	err = json.Unmarshal(body, addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if addTreatmentsResponse.TreatmentIds == nil || len(addTreatmentsResponse.TreatmentIds) == 0 {
		t.Fatal("Treatment ids expected to be returned for the treatments just added")
	}

	for _, treatmentId := range addTreatmentsResponse.TreatmentIds {
		if treatmentId == "" {
			t.Fatal("Treatment Id for the treatment added should not be empty")
		}
	}

	// get back the treatments for this patient visit to ensure that it is the same as what was passed in
	resp, err = http.Get(ts.URL + "?patient_visit_id=" + strconv.FormatInt(PatientVisitId, 10))
	if err != nil {
		t.Fatal("Unable to get treatments for patient visit " + err.Error())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the response made to get all treatments pertaining to patient visit " + err.Error())
	}

	CheckSuccessfulStatusCode(resp, "Unsuccessful call made to get treatments for patient visit: "+string(body), t)

	getTreatmentsResponse := &apiservice.GetTreatmentsResponse{}
	err = json.Unmarshal(body, getTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response body into the getTreatmentsResponse object " + err.Error())
	}

	if getTreatmentsResponse.Treatments == nil || len(getTreatmentsResponse.Treatments) == 0 {
		t.Fatal("Expected to get back treatments but got none")
	}

	return getTreatmentsResponse
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
