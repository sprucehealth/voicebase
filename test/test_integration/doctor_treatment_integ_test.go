package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/test"
)

func TestNewTreatmentSelection(t *testing.T) {
	t.Skip("Flakey test. Probably DoseSpot's fault.")

	testData := SetupTest(t)
	defer testData.Close(t)
	// use a real dosespot service before instantiating the server
	testData.Config.ERxAPI = testData.ERxAPI
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	cli := DoctorClient(testData, t, doctorID)

	res, err := cli.SelectMedication("Lisinopril (oral - tablet)", "10 mg")
	test.OK(t, err)
	if res.Treatment == nil {
		t.Fatal("Expected medication object to be populated but its not")
	}
	if res.Treatment.DrugDBIDs == nil || len(res.Treatment.DrugDBIDs) == 0 {
		t.Fatal("Expected additional drug db ids to be returned from api but none were")
	}
	if res.Treatment.DrugDBIDs[erx.LexiDrugSynID] == "0" || res.Treatment.DrugDBIDs[erx.LexiSynonymTypeID] == "0" || res.Treatment.DrugDBIDs[erx.LexiGenProductID] == "0" {
		t.Fatal("Expected additional drug db ids not set (lexi_drug_syn_id and lexi_synonym_type_id")
	}

	// Let's run a test for an OTC product to ensure that the OTC flag is set as expected
	res, err = cli.SelectMedication("Fish Oil (oral - capsule)", "500 mg")
	test.OK(t, err)
	if res.Treatment == nil || res.Treatment.OTC == false {
		t.Fatal("Expected the medication object to be returned and for the medication returned to be an OTC product")
	}

	// Let's ensure that we are returning a bad request to the doctor if they select a controlled substance
	_, err = cli.SelectMedication("Testosterone Cypionate (compounding - powder)", "cypionate")
	test.Equals(t, false, err == nil)

	// Let's ensure that we are rejecting a drug description that is longer than 105 characters to be routed via eRX.
	_, err = cli.SelectMedication(
		"Clinimix E Sulfite-Free 2.75% with 10% Dextrose and Electrolytes (intravenous - solution)",
		"Amino Acids 2.75% with 10% Dextrose and Electrolytes (Clinimix E Sulfite-Free)")
	test.Equals(t, false, err == nil)
}

func TestDispenseUnitIds(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	// use a real dosespot service before instantiating the server
	testData.Config.ERxAPI = testData.ERxAPI
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorMedicationDispenseUnitsURLPath, doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make a successful query to the medication dispense units api: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response: " + err.Error())
	}

	medicationDispenseUnitsResponse := &doctor_treatment_plan.MedicationDispenseUnitsResponse{}
	err = json.Unmarshal(body, medicationDispenseUnitsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal the response from the medication strength search api into a json object as expected: " + err.Error())
	}

	if medicationDispenseUnitsResponse.DispenseUnits == nil || len(medicationDispenseUnitsResponse.DispenseUnits) == 0 {
		t.Fatal("Expected dispense unit ids to be returned from api but none were")
	}

	for _, dispenseUnitItem := range medicationDispenseUnitsResponse.DispenseUnits {
		if dispenseUnitItem.ID == 0 || dispenseUnitItem.Text == "" {
			t.Fatal("Dispense Unit item was empty when this is not expected")
		}
	}

}

func TestAddTreatments(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// get the current primary doctor
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// doctor now attempts to add a couple treatments for patient
	treatment1 := &common.Treatment{
		DrugInternalName: "Drug1 (Route1 - Form1)",
		TreatmentPlanID:  treatmentPlan.ID,
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient instructions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatment2 := &common.Treatment{
		DrugInternalName: "Drug2 (Route2 - Form2)",
		TreatmentPlanID:  treatmentPlan.ID,
		DosageStrength:   "Strength2",
		DispenseValue:    2,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(27),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 3,
		},
		SubstitutionsAllowed: false,
		DaysSupply:           encoding.NullInt64{}, OTC: false,
		PharmacyNotes:       "testing pharmacy notes 2",
		PatientInstructions: "patient instructions 2",
		DrugDBIDs: map[string]string{
			"drug_db_id_3": "12414",
			"drug_db_id_4": "214",
		},
	}

	treatments := []*common.Treatment{treatment1, treatment2}

	getTreatmentsResponse := AddAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	for _, treatment := range getTreatmentsResponse.TreatmentList.Treatments {
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
	getTreatmentsResponse = AddAndGetTreatmentsForPatientVisit(testData, treatments, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// there should be just one treatment and its name should be the name that we just set
	if len(getTreatmentsResponse.TreatmentList.Treatments) != 1 {
		t.Fatal("Expected just 1 treatment to be returned after update")
	}

	// the dispense value should be set to 10
	if getTreatmentsResponse.TreatmentList.Treatments[0].DispenseValue != 10 {
		t.Fatal("Expected the updated dispense value to be set to 10")
	}

}

// This test ensures a fallback to select the medication to the drug database
// provider in the event that the drug description is not found in our database
func TestAddTreatments_FallbackToSelectMedication(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dr, _, _ := SignupRandomTestDoctor(t, testData)
	cli := DoctorClient(testData, t, dr.DoctorID)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	treatment1 := &common.Treatment{
		DrugInternalName: "Drug10 (Route10 - Form10)",
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		OTC:                 true,
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient insturctions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	var name, strength string
	var count int
	stubERxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubERxAPI.SelectMedicationFunc = func(clinicianID int64, medicationName, medicationStrength string) (*erx.MedicationSelectResponse, error) {
		name = medicationName
		strength = medicationStrength
		count++
		return &erx.MedicationSelectResponse{
			DispenseUnitID:          treatment1.DispenseUnitID.Int64(),
			DispenseUnitDescription: treatment1.DispenseUnitDescription,
			MatchedDrugName:         treatment1.DrugInternalName,
			OTC:                     treatment1.OTC,
		}, nil
	}

	_, err = cli.AddTreatmentsToTreatmentPlan([]*common.Treatment{treatment1}, tp.ID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, count)
	test.Equals(t, treatment1.DrugInternalName, name)
	test.Equals(t, treatment1.DosageStrength, strength)
}

func TestTreatmentTemplates(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// get the current primary doctor
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// doctor now attempts to favorite a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Drug1 (Route1 - Form1)",
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		OTC:                 true,
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient insturctions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatmentTemplate := &common.DoctorTreatmentTemplate{
		Name:      "Favorite Treatment #1",
		Treatment: treatment1,
	}

	treatmentTemplatesRequest := &doctor_treatment_plan.DoctorTreatmentTemplatesRequest{
		TreatmentPlanID:    treatmentPlan.ID,
		TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate},
	}
	data, err := json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentTemplatesURLPath, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, resp.StatusCode)
	}

	treatmentTemplatesResponse := &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if treatmentTemplatesResponse.TreatmentTemplates == nil || len(treatmentTemplatesResponse.TreatmentTemplates) != 1 {
		t.Fatal("Expected 1 favorited treatment in response but got none")
	}

	if treatmentTemplatesResponse.TreatmentTemplates[0].Name != treatmentTemplate.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugName != "Drug1" ||
		treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugRoute != "Route1" ||
		treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugForm != "Form1" {
		t.Fatalf("Expected the drug internal name to have been broken into its components %s %s %s", treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugName,
			treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugRoute, treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugForm)
	}

	// also ensure that drug db ids is not null or empty
	if len(treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugDBIDs) != 2 {
		t.Fatalf("Expected 2 drug db ids to exist instead got %d", len(treatmentTemplatesResponse.TreatmentTemplates[0].Treatment.DrugDBIDs))
	}

	treatment2 := &common.Treatment{
		DrugInternalName: "Drug2 (Route2 - Form2)",
		DosageStrength:   "Strength2",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient instructions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatmentTemplate2 := &common.DoctorTreatmentTemplate{}
	treatmentTemplate2.Name = "Treatment Template #2"
	treatmentTemplate2.Treatment = treatment2

	treatmentTemplatesRequest.TreatmentTemplates[0] = treatmentTemplate2
	data, err = json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorTreatmentTemplatesURLPath, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, resp.StatusCode)
	}

	treatmentTemplatesResponse = &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	} else if treatmentTemplatesResponse.TreatmentTemplates == nil || len(treatmentTemplatesResponse.TreatmentTemplates) != 2 {
		t.Fatal("Expected 2 favorited treatments in response")
	} else if treatmentTemplatesResponse.TreatmentTemplates[0].Name != treatmentTemplate.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	} else if treatmentTemplatesResponse.TreatmentTemplates[1].Name != treatmentTemplate2.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	// lets go ahead and delete each of the treatments
	treatmentTemplatesRequest.TreatmentTemplates = treatmentTemplatesResponse.TreatmentTemplates
	treatmentTemplatesRequest.TreatmentPlanID = treatmentPlan.ID
	data, err = json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = testData.AuthDelete(testData.APIServer.URL+apipaths.DoctorTreatmentTemplatesURLPath, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, resp.StatusCode)
	}

	treatmentTemplatesResponse = &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if len(treatmentTemplatesResponse.TreatmentTemplates) != 0 {
		t.Fatal("Expected 1 favorited treatment after deleting the first one")
	}
}

func TestTreatmentTemplatesInContextOfPatientVisit(t *testing.T) {

	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// get the current primary doctor
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// create random patient
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// doctor now attempts to favorite a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Drug1 (Route1 - Form1)",
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient insturctions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatmentTemplate := &common.DoctorTreatmentTemplate{}
	treatmentTemplate.Name = "Favorite Treatment #1"
	treatmentTemplate.Treatment = treatment1

	treatmentTemplatesRequest := &doctor_treatment_plan.DoctorTreatmentTemplatesRequest{TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate}}
	treatmentTemplatesRequest.TreatmentPlanID = treatmentPlan.ID
	data, err := json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	treatmentTemplatesURL := testData.APIServer.URL + apipaths.DoctorTreatmentTemplatesURLPath
	resp, err := testData.AuthPost(treatmentTemplatesURL, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	treatmentTemplatesResponse := &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.Unmarshal(body, treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if treatmentTemplatesResponse.TreatmentTemplates == nil || len(treatmentTemplatesResponse.TreatmentTemplates) != 1 {
		t.Fatal("Expected 1 favorited treatment in response but got none")
	}

	if treatmentTemplatesResponse.TreatmentTemplates[0].Name != treatmentTemplate.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	treatment2 := &common.Treatment{
		DrugInternalName: "Drug2 (Route2 - Form2)",
		DosageStrength:   "Strength2",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient instructions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	// lets add this as a treatment to the patient visit
	getTreatmentsResponse := AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment2}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	if len(getTreatmentsResponse.TreatmentList.Treatments) != 1 {
		t.Fatal("Expected patient visit to have 1 treatment")
	}

	// now, lets favorite a treatment that exists for the patient visit
	treatmentTemplate2 := &common.DoctorTreatmentTemplate{}
	treatmentTemplate2.Name = "Favorite Treatment #2"
	treatmentTemplate2.Treatment = getTreatmentsResponse.TreatmentList.Treatments[0]
	treatmentTemplatesRequest.TreatmentTemplates[0] = treatmentTemplate2
	treatmentTemplatesRequest.TreatmentPlanID = treatmentPlan.ID

	data, err = json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp2, err := testData.AuthPost(treatmentTemplatesURL, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", resp2.StatusCode)
	}

	body, err = ioutil.ReadAll(resp2.Body)
	if err != nil {
		t.Fatal("Unable to read from response body: " + err.Error())
	}

	treatmentTemplatesResponse = &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.Unmarshal(body, treatmentTemplatesResponse)

	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if treatmentTemplatesResponse.TreatmentTemplates == nil || len(treatmentTemplatesResponse.TreatmentTemplates) != 2 {
		t.Fatal("Expected 2 favorited treatments in response")
	}

	if treatmentTemplatesResponse.TreatmentTemplates[0].Name != treatmentTemplate.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if treatmentTemplatesResponse.TreatmentTemplates[1].Name != treatmentTemplate2.Name {
		t.Fatal("Expected the same favorited treatment to be returned that was added")
	}

	if len(treatmentTemplatesResponse.TreatmentTemplates) == 0 {
		t.Fatal("Expected there to be 1 treatment added to the visit and the doctor")
	}

	if treatmentTemplatesResponse.Treatments[0].DoctorTreatmentTemplateID.Int64() != treatmentTemplatesResponse.TreatmentTemplates[1].ID.Int64() {
		t.Fatal("Expected the favoriteTreatmentId to be set for the treatment and to be set to the right treatment")
	}

	// now, lets go ahead and add a treatment to the patient visit from a favorite treatment
	treatment1.DoctorTreatmentTemplateID = encoding.DeprecatedNewObjectID(treatmentTemplatesResponse.TreatmentTemplates[0].ID.Int64())
	treatment2.DoctorTreatmentTemplateID = encoding.DeprecatedNewObjectID(treatmentTemplatesResponse.TreatmentTemplates[1].ID.Int64())
	getTreatmentsResponse = AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1, treatment2}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	if len(getTreatmentsResponse.TreatmentList.Treatments) != 2 {
		t.Fatal("There should exist 2 treatments for the patient visit")
	}

	if getTreatmentsResponse.TreatmentList.Treatments[0].DoctorTreatmentTemplateID.Int64() == 0 || getTreatmentsResponse.TreatmentList.Treatments[1].DoctorTreatmentTemplateID.Int64() == 0 {
		t.Fatal("Expected the doctorFavoriteId to be set for both treatments given that they were added from favorites")
	}

	treatmentTemplate.ID = encoding.DeprecatedNewObjectID(getTreatmentsResponse.TreatmentList.Treatments[0].DoctorTreatmentTemplateID.Int64())
	treatmentTemplate.Treatment = getTreatmentsResponse.TreatmentList.Treatments[0]
	treatmentTemplatesRequest.TreatmentTemplates = []*common.DoctorTreatmentTemplate{treatmentTemplate}
	treatmentTemplatesRequest.TreatmentPlanID = treatmentPlan.ID
	// lets delete a favorite that is also a treatment in the patient visit
	data, err = json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = testData.AuthDelete(treatmentTemplatesURL, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	treatmentTemplatesResponse = &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.NewDecoder(resp.Body).Decode(treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	if len(treatmentTemplatesResponse.TreatmentTemplates) != 1 {
		t.Fatal("Expected 1 favorited treatment after deleting the first one")
	}

	// ensure that treatments are still returned
	if len(treatmentTemplatesResponse.Treatments) != 2 {
		t.Fatal("Expected there to exist 2 treatments for the patient visit even after deleting one of the treatments")
	}

	if treatmentTemplatesResponse.Treatments[0].DoctorTreatmentTemplateID.Int64() != 0 {
		t.Fatal("Expected the first treatment to no longer be a favorited treatment")
	}
}

func TestTreatmentTemplateWithDrugOutOfMarket(t *testing.T) {
	t.Skip("Skipping test for now until long term fix as defined in https://github.com/SpruceHealth/backend/pull/657 is resolved.")
	testData := SetupTest(t)
	defer testData.Close(t)
	// use a real dosespot service before instantiating the server
	testData.Config.ERxAPI = testData.ERxAPI
	testData.StartAPIServer(t)

	// get the current primary doctor
	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	// doctor now attempts to favorite a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Drug1 (Route1 - Form1)",
		DosageStrength:   "Strength1",
		DispenseValue:    1,
		DispenseUnitID:   encoding.DeprecatedNewObjectID(26),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		SubstitutionsAllowed: true,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		},
		PharmacyNotes:       "testing pharmacy notes",
		PatientInstructions: "patient insturctions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}

	treatmentTemplate := &common.DoctorTreatmentTemplate{
		Name:      "Favorite Treatment #1",
		Treatment: treatment1,
	}

	treatmentTemplatesURL := testData.APIServer.URL + apipaths.DoctorTreatmentTemplatesURLPath

	treatmentTemplatesRequest := &doctor_treatment_plan.DoctorTreatmentTemplatesRequest{
		TreatmentPlanID:    treatmentPlan.ID,
		TreatmentTemplates: []*common.DoctorTreatmentTemplate{treatmentTemplate},
	}
	data, err := json.Marshal(&treatmentTemplatesRequest)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := testData.AuthPost(treatmentTemplatesURL, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 but got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to read body of the post request made to add treatments to patient visit: " + err.Error())
	}

	treatmentTemplatesResponse := &doctor_treatment_plan.DoctorTreatmentTemplatesResponse{}
	err = json.Unmarshal(body, treatmentTemplatesResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	// lets' attempt to add the favorited treatment to a patient visit. It should fail because the stubErxApi is wired
	// to return no medication to indicate drug is no longer in market
	treatment1.DoctorTreatmentTemplateID = treatmentTemplatesResponse.TreatmentTemplates[0].ID
	treatmentRequestBody := doctor_treatment_plan.AddTreatmentsRequestBody{
		TreatmentPlanID: treatmentPlan.ID,
		Treatments:      []*common.Treatment{treatment1},
	}

	treatmentsURL := testData.APIServer.URL + apipaths.DoctorVisitTreatmentsURLPath
	data, err = json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err = testData.AuthPost(treatmentsURL, "application/json", bytes.NewBuffer(data), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to add treatments to patient visit: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected the call to add treatments to error out with bad request (400) because treatment is out of market, but instead got %d returned", resp.StatusCode)
	}
}

func compareTreatments(treatment *common.Treatment, treatment1 *common.Treatment, t *testing.T) {
	if treatment.DosageStrength != treatment1.DosageStrength || treatment.DispenseValue != treatment1.DispenseValue ||
		treatment.DispenseUnitID.Int64() != treatment1.DispenseUnitID.Int64() || treatment.PatientInstructions != treatment1.PatientInstructions ||
		treatment.PharmacyNotes != treatment1.PharmacyNotes || treatment.NumberRefills != treatment1.NumberRefills ||
		treatment.SubstitutionsAllowed != treatment1.SubstitutionsAllowed || treatment.DaysSupply != treatment1.DaysSupply ||
		treatment.OTC != treatment1.OTC {
		treatmentData, _ := json.MarshalIndent(treatment, "", " ")
		treatment1Data, _ := json.MarshalIndent(treatment1, "", " ")

		t.Fatalf("Treatment returned from the call to get treatments for patient visit not the same as what was added for the patient visit: treatment returned: %s, treatment added: %s", string(treatmentData), string(treatment1Data))
	}
}
