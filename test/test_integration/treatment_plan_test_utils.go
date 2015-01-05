package test_integration

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/test"
)

func GetRegimenPlanForTreatmentPlan(testData *TestData, doctor *common.Doctor, treatmentPlanID int64, t *testing.T) *common.RegimenPlan {
	resp, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanID, 10), doctor.AccountID.Int64())
	if err != nil {
		t.Fatal("Unable to get regimen for patient visit: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Unable to parse the body of the response for getting the regimen plan: " + err.Error())
	}

	doctorTreatmentPlanResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.Unmarshal(body, doctorTreatmentPlanResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal body into json object: " + err.Error())
	}

	return doctorTreatmentPlanResponse.TreatmentPlan.RegimenPlan
}

func CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan, testData *TestData, doctor *common.Doctor, t *testing.T) *common.RegimenPlan {
	// TODO: replace instance of this function with the few lines below
	cli := DoctorClient(testData, t, doctor.DoctorID.Int64())
	rp, err := cli.CreateRegimenPlan(regimenPlan)
	if err != nil {
		t.Fatalf("Failed to create regimen plan: %s [%s]", err.Error(), CallerString(1))
	}
	return rp
}

func GetListOfTreatmentPlansForPatient(patientID, doctorAccountID int64, testData *TestData, t *testing.T) *doctor_treatment_plan.TreatmentPlansResponse {
	response := &doctor_treatment_plan.TreatmentPlansResponse{}
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorTreatmentPlansListURLPath+"?patient_id="+strconv.FormatInt(patientID, 10), doctorAccountID)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}

	return response
}

func DeleteTreatmentPlanForDoctor(treatmentPlanID, doctorAccountID int64, testData *TestData, t *testing.T) {
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanID: treatmentPlanID,
	})

	res, err := testData.AuthDelete(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctorAccountID)
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d but got %d instead", http.StatusOK, res.StatusCode)
	}
}

func GetDoctorTreatmentPlanByID(treatmentPlanID, doctorAccountID int64, testData *TestData, t *testing.T) *common.TreatmentPlan {
	response := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath+"?treatment_plan_id="+strconv.FormatInt(treatmentPlanID, 10), doctorAccountID)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d but got %d instead", http.StatusOK, res.StatusCode)
	} else if err := json.NewDecoder(res.Body).Decode(response); err != nil {
		t.Fatalf(err.Error())
	}
	doctor, err := testData.DataAPI.GetDoctorFromAccountID(doctorAccountID)
	if err != nil {
		t.Fatal(err)
	}
	role := api.DOCTOR_ROLE
	if doctor.IsMA {
		role = api.MA_ROLE
	}
	tp, err := doctor_treatment_plan.TransformTPFromResponse(testData.DataAPI, response.TreatmentPlan, doctor.DoctorID.Int64(), role)
	if err != nil {
		t.Fatal(err)
	}
	return tp
}

func AddAndGetTreatmentsForPatientVisit(testData *TestData, treatments []*common.Treatment, doctorAccountID, treatmentPlanID int64, t *testing.T) *doctor_treatment_plan.GetTreatmentsResponse {
	testData.Config.ERxAPI = &erx.StubErxService{
		SelectedMedicationToReturn: &erx.MedicationSelectResponse{},
	}

	treatmentRequestBody := doctor_treatment_plan.AddTreatmentsRequestBody{
		TreatmentPlanID: encoding.NewObjectID(treatmentPlanID),
		Treatments:      treatments,
	}

	data, err := json.Marshal(&treatmentRequestBody)
	if err != nil {
		t.Fatal("Unable to marshal request body for adding treatments to patient visit")
	}

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorVisitTreatmentsURLPath, "application/json", bytes.NewBuffer(data), doctorAccountID)
	if err != nil {
		t.Fatal("Unable to make POST request to add treatments to patient visit " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 instead got %d [%s]", resp.StatusCode, CallerString(1))
	}

	addTreatmentsResponse := &doctor_treatment_plan.GetTreatmentsResponse{}
	err = json.NewDecoder(resp.Body).Decode(addTreatmentsResponse)
	if err != nil {
		t.Fatal("Unable to unmarshal response into object : " + err.Error())
	}

	treatmentList := &common.TreatmentList{Treatments: treatments}
	if !treatmentList.Equals(addTreatmentsResponse.TreatmentList) {
		t.Fatal("Expected treatments added to match treatments returned but they dont")
	}

	return addTreatmentsResponse
}

func ValidateRegimenRequestAgainstResponse(doctorRegimenRequest, doctorRegimenResponse *common.RegimenPlan, t *testing.T) {

	// there should be the same number of sections in the request and the response
	if len(doctorRegimenRequest.Sections) != len(doctorRegimenResponse.Sections) {
		t.Fatalf("Number of regimen sections should be the same in the request and the response. Request = %d, response = %d", len(doctorRegimenRequest.Sections), len(doctorRegimenResponse.Sections))
	}

	// there should be the same number of steps in each section in the request and the response
	if doctorRegimenRequest.Sections != nil {
		for i, regimenSection := range doctorRegimenRequest.Sections {
			if len(regimenSection.Steps) != len(doctorRegimenResponse.Sections[i].Steps) {
				t.Fatalf(`the number of regimen steps in the regimen section of the request and the response should be the same, 
				regimen section = %s, request = %d, response = %d`, regimenSection.Name, len(regimenSection.Steps), len(doctorRegimenResponse.Sections[i].Steps))
			}
		}
	}

	// the number of steps in each regimen section should be the same across the request and response
	for i, regimenSection := range doctorRegimenRequest.Sections {
		if len(regimenSection.Steps) != len(doctorRegimenResponse.Sections[i].Steps) {
			t.Fatalf("Expected have the same number of regimen steps for each section. Section %s has %d steps but expected %d steps", regimenSection.Name, len(regimenSection.Steps), len(doctorRegimenResponse.Sections[i].Steps))
		}
	}

	// all regimen steps should have an id in the response
	regimenStepsMapping := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if regimenStep.ID.Int64() == 0 {
			t.Fatal("Regimen steps in the response are expected to have an id")
		}
		regimenStepsMapping[regimenStep.ID.Int64()] = true
	}

	// all regimen steps in the regimen sections should have an id in the response
	// all regimen steps in the sections that have a parentId should also be present in the global list
	for _, regimenSection := range doctorRegimenResponse.Sections {
		for _, regimenStep := range regimenSection.Steps {
			if regimenStep.ID.Int64() == 0 {
				t.Fatal("Regimen steps in each section are expected to have an id")
			}
			if regimenStep.ParentID.IsValid && regimenStepsMapping[regimenStep.ParentID.Int64()] == false {
				t.Fatalf("There exists a regimen step in a section that is not present in the global list. Id of regimen step %d", regimenStep.ID.Int64Value)
			}
		}
	}

	// no two items should have the same id
	idsFound := make(map[int64]bool)
	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if _, ok := idsFound[regimenStep.ID.Int64()]; ok {
			t.Fatal("No two items can have the same id in the global list")
		}
		idsFound[regimenStep.ID.Int64()] = true
	}

	// deleted regimen steps should not show up in the response
	deletedRegimenStepIds := make(map[int64]bool)
	// updated regimen steps should have a different id in the response
	updatedRegimenSteps := make(map[string][]int64)

	for _, regimenStep := range doctorRegimenRequest.AllSteps {
		switch regimenStep.State {
		case common.STATE_MODIFIED:
			updatedRegimenSteps[regimenStep.Text] = append(updatedRegimenSteps[regimenStep.Text], regimenStep.ID.Int64())
		}
	}

	for _, regimenStep := range doctorRegimenResponse.AllSteps {
		if updatedIds, ok := updatedRegimenSteps[regimenStep.Text]; ok {
			for _, updatedID := range updatedIds {
				if regimenStep.ID.Int64() == updatedID {
					t.Fatalf("Expected an updated regimen step to have a different id in the response. Id = %d", regimenStep.ID.Int64())
				}
			}
		}

		if deletedRegimenStepIds[regimenStep.ID.Int64()] == true {
			t.Fatalf("Expected regimen step %d to have been deleted and not in the response", regimenStep.ID.Int64())
		}
	}
}

func CreateFavoriteTreatmentPlan(treatmentPlanID int64, testData *TestData, doctor *common.Doctor, t *testing.T) *doctor_treatment_plan.FavoriteTreatmentPlan {
	cli := DoctorClient(testData, t, doctor.DoctorID.Int64())

	// lets submit a regimen plan for this patient
	// reason we do this is because the regimen steps have to exist before treatment plan can be favorited,
	// and the only way we can create regimen steps today is in the context of a patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: encoding.NewObjectID(treatmentPlanID),
	}

	regimenStep1 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 1",
		State: common.STATE_ADDED,
	}

	regimenStep2 := &common.DoctorInstructionItem{
		Text:  "Regimen Step 2",
		State: common.STATE_ADDED,
	}

	regimenSection := &common.RegimenSection{
		Name: "morning",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep1.Text,
			State: common.STATE_ADDED,
		}},
	}

	regimenSection2 := &common.RegimenSection{
		Name: "night",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep2.Text,
			State: common.STATE_ADDED,
		}},
	}

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse, err := cli.CreateRegimenPlan(regimenPlanRequest)
	if err != nil {
		t.Fatalf("Failed to create regimen: %s [%s]", err.Error(), CallerString(1))
	}
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// prepare the regimen steps and the advice points to be added into the sections
	// after the global list for each has been updated to include items.
	// the reason this is important is because favorite treatment plans require items to exist that are linked
	// from the master list
	regimenSection.Steps[0].ParentID = regimenPlanResponse.AllSteps[0].ID
	regimenSection2.Steps[0].ParentID = regimenPlanResponse.AllSteps[1].ID

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &doctor_treatment_plan.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		Note: "FTP Note",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{{
				DrugDBIDs: map[string]string{
					erx.LexiDrugSynID:     "1234",
					erx.LexiGenProductID:  "12345",
					erx.LexiSynonymTypeID: "123556",
					erx.NDC:               "2415",
				},
				DrugInternalName:        "Teting (This - Drug)",
				DosageStrength:          "10 mg",
				DispenseValue:           5,
				DispenseUnitDescription: "Tablet",
				DispenseUnitID:          encoding.NewObjectID(19),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
				SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
				PatientInstructions: "Take once daily",
				OTC:                 false,
			}},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: regimenPlanResponse.AllSteps,
			Sections: []*common.RegimenSection{regimenSection, regimenSection2},
		},
	}

	ftp, err := cli.CreateFavoriteTreatmentPlan(favoriteTreatmentPlan)
	if err != nil {
		t.Fatalf("Failed to create ftp: %s [%s]", err.Error(), CallerString(1))
	}

	if ftp.RegimenPlan == nil || len(ftp.RegimenPlan.Sections) != 2 {
		t.Fatalf("Expected to have a regimen plan or 2 items in the regimen section [%s]", CallerString(1))
	}

	return ftp
}
