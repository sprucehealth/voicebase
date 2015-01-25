package test_treatment_plan

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that treatment plans can be versioned
// and that the content source and the parent are created as expected
func TestVersionTreatmentPlan_NewTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	// get patient to start a visit and doctor to pick treatment plan
	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// submit treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan that is a version of the previous one
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	if tpResponse.TreatmentPlan.ID.Int64() == treatmentPlan.ID.Int64() {
		t.Fatal("Expected treatment plan to be different given that it was just versioned")
	}

	currentTreatmentPlan, err := testData.DataAPI.GetTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID)
	test.OK(t, err)

	// the first treatment plan should be the parent of this treatment plan
	if currentTreatmentPlan.Parent.ParentType != common.TPParentTypeTreatmentPlan ||
		currentTreatmentPlan.Parent.ParentID.Int64() != treatmentPlan.ID.Int64() {
		t.Fatalf("expected treatment plan id %d to be the parent of treatment plan id %d but it wasnt", treatmentPlan.ID.Int64(), currentTreatmentPlan.ID.Int64())
	}

	// there should be no content source for this treatment plan
	if currentTreatmentPlan.ContentSource != nil {
		t.Fatal("Expected no content source for this treatment plan")
	}

	// there should be no treatments, regimen or advice
	if len(currentTreatmentPlan.TreatmentList.Treatments) > 0 {
		t.Fatalf("Expected no treatments isntead got %d", len(currentTreatmentPlan.TreatmentList.Treatments))
	} else if len(currentTreatmentPlan.RegimenPlan.Sections) > 0 {
		t.Fatalf("Expected no regimen sections instead got %d", len(currentTreatmentPlan.RegimenPlan.Sections))
	}

	// should get back 1 treatment plan in draft and the other one active
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plan in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	// now go ahead and submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(currentTreatmentPlan.ID.Int64(), doctor, testData, t)

	// the new versioned treatment plan should be active and the previous one inactice
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected 0 treamtent plans in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if treatmentPlanResponse.ActiveTreatmentPlans[0].ID.Int64() != currentTreatmentPlan.ID.Int64() {
		t.Fatalf("Expected treatment plan id %d instead got %d", currentTreatmentPlan.ID.Int64(), treatmentPlanResponse.ActiveTreatmentPlans[0].ID.Int64())
	} else if len(treatmentPlanResponse.InactiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 inactive treatment plan instead got %d", len(treatmentPlanResponse.InactiveTreatmentPlans))
	} else if treatmentPlanResponse.InactiveTreatmentPlans[0].ID.Int64() != treatmentPlan.ID.Int64() {
		t.Fatalf("Expected inactive treatment plan to be %d instead it was %d", treatmentPlan.ID.Int64(), treatmentPlanResponse.InactiveTreatmentPlans[0].ID.Int64())
	}
}

// This test is to ensure that we can start with a previous treatment plan
// when versioning a treatment plan
func TestVersionTreatmentPlan_PrevTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// add treatments
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil (oral - tablet)",
		DrugRoute:        "oral",
		DrugForm:         "tablet",
		TreatmentPlanID:  treatmentPlan.ID,
		DosageStrength:   "10 mg",
		DispenseValue:    1,
		DispenseUnitID:   encoding.NewObjectID(26),
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
		PatientInstructions: "patient instructions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// add regimen steps
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanID = treatmentPlan.ID
	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED
	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
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
	test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan that is a version of the previous one
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeTreatmentPlan,
		ID:   treatmentPlan.ID,
	}, doctor, testData, t)

	if tpResponse.TreatmentPlan.ID.Int64() == treatmentPlan.ID.Int64() {
		t.Fatal("Expected treatment plan to be different given that it was just versioned")
	}

	// this treatment plan should have the same contents as the treatment plan picked
	// as the content source
	if err != nil {
		t.Fatal(err)
	} else if len(tpResponse.TreatmentPlan.TreatmentList.Treatments) != 1 {
		t.Fatalf("Expected 1 treatment instead got %d", len(tpResponse.TreatmentPlan.TreatmentList.Treatments))
	} else if tpResponse.TreatmentPlan.TreatmentList.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the treatment list to be uncommitted but it wasnt")
	} else if len(tpResponse.TreatmentPlan.RegimenPlan.Sections) != 2 {
		t.Fatalf("Expected 2 regimen sections instead got %d", len(tpResponse.TreatmentPlan.RegimenPlan.Sections))
	} else if tpResponse.TreatmentPlan.RegimenPlan.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the regimen plan to be uncommitted but it wasnt")
	}

	// ensure that the content source is the treatment plan
	if tpResponse.TreatmentPlan.ContentSource == nil ||
		tpResponse.TreatmentPlan.ContentSource.Type != common.TPContentSourceTypeTreatmentPlan {
		t.Fatalf("Expected the content source to be treatment plan but it wasnt")
	} else if tpResponse.TreatmentPlan.ContentSource.Deviated {
		t.Fatal("Didn't expect the treatment plan to deviate from the content source yet")
	}

	// now try to modify the treatment and it should mark the treatment plan as having deviated from the source
	treatment1.DispenseValue = encoding.HighPrecisionFloat64(21151)
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), tpResponse.TreatmentPlan.ID.Int64(), t)

	currentTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID)
	test.OK(t, err)

	if !currentTreatmentPlan.ContentSource.HasDeviated {
		t.Fatal("Expected the treatment plan to have deviated from the content source but it hasnt")
	}
}

// This test is to ensure that we can create multiple versions of treatment plans
// and submit them with no problem
func TestVersionTreatmentPlan_MultipleRevs(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan from scratch
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// add treatments
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil (oral - tablet)",
		DrugRoute:        "oral",
		DrugForm:         "tablet",
		TreatmentPlanID:  tpResponse.TreatmentPlan.ID,
		DosageStrength:   "10 mg",
		DispenseValue:    1,
		DispenseUnitID:   encoding.NewObjectID(26),
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
		PatientInstructions: "patient instructions",
		DrugDBIDs: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), tpResponse.TreatmentPlan.ID.Int64(), t)

	// add regimen steps
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanID = tpResponse.TreatmentPlan.ID
	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED
	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
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
	test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(tpResponse.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// start yet another treatment plan, this time from the previous treatment plan
	tpResponse2 := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tpResponse.TreatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeTreatmentPlan,
		ID:   tpResponse.TreatmentPlan.ID,
	}, doctor, testData, t)

	parentTreatmentPlan, err := testData.DataAPI.GetTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID)
	test.OK(t, err)

	tp2, err := responses.TransformTPFromResponse(testData.DataAPI, tpResponse2.TreatmentPlan, doctor.DoctorID.Int64(), api.DOCTOR_ROLE)
	if err != nil {
		t.Fatal(err)
	}

	if !parentTreatmentPlan.Equals(tp2) {
		t.Fatal("Expected the parent and the newly versioned treatment plan to be equal but they are not")
	}

	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plans in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if len(treatmentPlanResponse.InactiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 inactive treatment plan instead got %d", len(treatmentPlanResponse.InactiveTreatmentPlans))
	}
}

// This test is to ensure that we don't allow versioning from an inactive treatment plan
func TestVersionTreatmentPlan_PickingFromInactiveTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan from scratch
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(tpResponse.TreatmentPlan.ID.Int64(), doctor, testData, t)

	// attempt to start yet another treatment plan but this time trying to pick from
	// an inactive treatment plan. this should fail

	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentID:   treatmentPlan.ID,
			ParentType: common.TPParentTypeTreatmentPlan,
		},
	})

	res, err := testData.AuthPut(testData.APIServer.URL+apipaths.DoctorTreatmentPlansURLPath, "application/json", bytes.NewReader(jsonData), doctor.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected %d but got %d", http.StatusBadRequest, res.StatusCode)
	}

}

// This test is to ensure that doctor can pick from a favorite treatment plan to
// version a treatment plan
func TestVersionTreatmentPlan_PickFromFTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := test_integration.CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan from an FTP
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   favoriteTreatmentPlan.ID,
	}, doctor, testData, t)

	if !favoriteTreatmentPlan.EqualsTreatmentPlan(tpResponse.TreatmentPlan) {
		t.Fatal("Expected contents of favorite treatment plan to be the same as that of the treatment plan")
	}
}

// This test is to ensure that the most active treatment plan is shared with the patient
func TestVersionTreatmentPlan_TPForPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatal(err)
	}

	// version treatment plan
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// submit version to make it active
	test_integration.SubmitPatientVisitBackToPatient(tpResponse.TreatmentPlan.ID.Int64(), doctor, testData, t)

	tps, err := testData.DataAPI.GetActiveTreatmentPlansForPatient(patientID)
	test.OK(t, err)
	test.Equals(t, 1, len(tps))
	treatmentPlanForPatient := tps[0]

	if treatmentPlanForPatient.ID.Int64() != tpResponse.TreatmentPlan.ID.Int64() {
		t.Fatal("Expected the latest treatment plan to be the one considered active for patient but it wasnt the case")
	}
}

// This test is to ensure that we don't deviate the treatment plan
// unless the data has actually changed
func TestVersionTreatmentPlan_DeviationFromFTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	cli := test_integration.DoctorClient(testData, t, doctorID)

	// get patient to start a visit and doctor to pick treatment plan
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := test_integration.CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now try to start a new treatment plan from an FTP
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   favoriteTreatmentPlan.ID,
	}, doctor, testData, t)

	// now, submit the exact same treatments to commit it
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, favoriteTreatmentPlan.TreatmentList.Treatments, doctor.AccountID.Int64(), tpResponse.TreatmentPlan.ID.Int64(), t)

	currentTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID)
	if err != nil {
		t.Fatal(err)
	} else if currentTreatmentPlan.ContentSource.HasDeviated {
		t.Fatal("Did not expect treatment plan to deviate from source but it did")
	}

	// submit the exact same regimen
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanID = tpResponse.TreatmentPlan.ID
	regimenPlanRequest.Sections = favoriteTreatmentPlan.RegimenPlan.Sections
	test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	currentTreatmentPlan, err = testData.DataAPI.GetAbridgedTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID)
	if err != nil {
		t.Fatal(err)
	} else if currentTreatmentPlan.ContentSource.HasDeviated {
		t.Fatal("Did not expect treatment plan to deviate from source but it did")
	}

	// changing note should deviate FTP
	if err := cli.UpdateTreatmentPlanNote(tpResponse.TreatmentPlan.ID.Int64(), "something else"); err != nil {
		t.Fatal(err)
	}
	if tp, err := testData.DataAPI.GetAbridgedTreatmentPlan(tpResponse.TreatmentPlan.ID.Int64(), doctorID); err != nil {
		t.Fatal(err)
	} else if !tp.ContentSource.HasDeviated {
		t.Fatal("Expected treatment plan to deviate when changing note")
	}
}

func TestVersionTreatmentPlan_DeleteOlderDraft(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}
	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// attempt to version treatment plan
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// attempt to version again
	tpResponse2 := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	// two treatment plans should be different given that older one should be deleted
	if tpResponse.TreatmentPlan.ID.Int64() == tpResponse2.TreatmentPlan.ID.Int64() {
		t.Fatal("Expected a new treatment plan to be created if the user attempts to pick again")
	}

	// attempt to create FTP under the new versioned treatment plan
	favoriteTreatmentPlan := test_integration.CreateFavoriteTreatmentPlan(tpResponse2.TreatmentPlan.ID.Int64(), testData, doctor, t)

	// attempt to start a new TP now with this FTP
	tpResponse3 := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   treatmentPlan.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   favoriteTreatmentPlan.ID,
	}, doctor, testData, t)

	if tpResponse3.TreatmentPlan.ID.Int64() == tpResponse2.TreatmentPlan.ID.Int64() {
		t.Fatal("Expected the newly created treatment plan to have a different id than the previous one")
	}

	// there should only exist 1 draft and 1 active treatment plan
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plans in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if len(treatmentPlanResponse.InactiveTreatmentPlans) != 0 {
		t.Fatalf("Expected 0 inactive treatment plans instead got %d", len(treatmentPlanResponse.InactiveTreatmentPlans))
	}

}
