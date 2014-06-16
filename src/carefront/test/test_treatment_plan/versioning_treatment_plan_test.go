package test_treatment_plan

import (
	"carefront/api"
	"carefront/common"
	"carefront/encoding"
	"carefront/test/test_integration"
	"testing"
)

func TestVersionTreatmentPlan_NewTP(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	patientVisitResponse, treatmentPlan := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	patientId, err := testData.DataApi.GetPatientIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// submit treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// now try to start a new treatment plan that is a version of the previous one
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   treatmentPlan.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)

	if tpResponse.TreatmentPlan.Id.Int64() == treatmentPlan.Id.Int64() {
		t.Fatal("Expected treatment plan to be different given that it was just versioned")
	}

	currentTreatmentPlan, err := testData.DataApi.GetTreatmentPlan(tpResponse.TreatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatal(err)
	}

	// the first treatment plan should be the parent of this treatment plan
	if currentTreatmentPlan.Parent.ParentType != common.TPParentTypeTreatmentPlan ||
		currentTreatmentPlan.Parent.ParentId.Int64() != treatmentPlan.Id.Int64() {
		t.Fatalf("expected treatment plan id %d to be the parent of treatment plan id %d but it wasnt", treatmentPlan.Id.Int64(), currentTreatmentPlan.Id.Int64())
	}

	// there should be no content source for this treatment plan
	if currentTreatmentPlan.ContentSource != nil {
		t.Fatal("Expected no content source for this treatment plan")
	}

	// there should be no treatments, regimen or advice
	if len(currentTreatmentPlan.TreatmentList.Treatments) > 0 {
		t.Fatalf("Expected no treatments isntead got %d", len(currentTreatmentPlan.TreatmentList.Treatments))
	} else if len(currentTreatmentPlan.RegimenPlan.RegimenSections) > 0 {
		t.Fatalf("Expected no regimen sections instead got %d", len(currentTreatmentPlan.RegimenPlan.RegimenSections))
	} else if len(currentTreatmentPlan.Advice.SelectedAdvicePoints) > 0 {
		t.Fatalf("Expected no advice points instead got %d", len(currentTreatmentPlan.Advice.SelectedAdvicePoints))
	}

	// should get back 1 treatment plan in draft and the other one active
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plan in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	// now go ahead and submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(currentTreatmentPlan.Id.Int64(), doctor, testData, t)

	// the new versioned treatment plan should be active and the previous one inactice
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientId, doctor.AccountId.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected 0 treamtent plans in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if treatmentPlanResponse.ActiveTreatmentPlans[0].Id.Int64() != currentTreatmentPlan.Id.Int64() {
		t.Fatalf("Expected treatment plan id %d instead got %d", currentTreatmentPlan.Id.Int64(), treatmentPlanResponse.ActiveTreatmentPlans[0].Id.Int64())
	} else if len(treatmentPlanResponse.InactiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 inactive treatment plan instead got %d", len(treatmentPlanResponse.InactiveTreatmentPlans))
	} else if treatmentPlanResponse.InactiveTreatmentPlans[0].Id.Int64() != treatmentPlan.Id.Int64() {
		t.Fatalf("Expected inactive treatment plan to be %d instead it was %d", treatmentPlan.Id.Int64(), treatmentPlanResponse.InactiveTreatmentPlans[0].Id.Int64())
	}
}

func TestVersionTreatmentPlan_PrevTP(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)
	doctorId := test_integration.GetDoctorIdOfCurrentPrimaryDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorId)
	if err != nil {
		t.Fatal("Unable to get doctor from doctor id " + err.Error())
	}

	// get patient to start a visit and doctor to pick treatment plan
	_, treatmentPlan := test_integration.SignupAndSubmitPatientVisitForRandomPatient(t, testData, doctor)

	// add treatments
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil",
		TreatmentPlanId:  treatmentPlan.Id,
		DosageStrength:   "10 mg",
		DispenseValue:    1,
		DispenseUnitId:   encoding.NewObjectId(26),
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
		DrugDBIds: map[string]string{
			"drug_db_id_1": "12315",
			"drug_db_id_2": "124",
		},
	}
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), treatmentPlan.Id.Int64(), t)

	// add advice
	advicePoint1 := &common.DoctorInstructionItem{Text: "Advice point 1", State: common.STATE_ADDED}
	advicePoint2 := &common.DoctorInstructionItem{Text: "Advice point 2", State: common.STATE_ADDED}
	doctorAdviceRequest := &common.Advice{}
	doctorAdviceRequest.AllAdvicePoints = []*common.DoctorInstructionItem{advicePoint1, advicePoint2}
	doctorAdviceRequest.SelectedAdvicePoints = doctorAdviceRequest.AllAdvicePoints
	doctorAdviceRequest.TreatmentPlanId = treatmentPlan.Id
	test_integration.UpdateAdvicePointsForPatientVisit(doctorAdviceRequest, testData, doctor, t)

	// add regimen steps
	regimenPlanRequest := &common.RegimenPlan{}
	regimenPlanRequest.TreatmentPlanId = treatmentPlan.Id
	regimenStep1 := &common.DoctorInstructionItem{}
	regimenStep1.Text = "Regimen Step 1"
	regimenStep1.State = common.STATE_ADDED
	regimenStep2 := &common.DoctorInstructionItem{}
	regimenStep2.Text = "Regimen Step 2"
	regimenStep2.State = common.STATE_ADDED
	regimenPlanRequest.AllRegimenSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenSection := &common.RegimenSection{}
	regimenSection.RegimenName = "morning"
	regimenSection.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenPlanRequest.AllRegimenSteps[0].Id,
		Text:     regimenPlanRequest.AllRegimenSteps[0].Text,
	},
	}
	regimenSection2 := &common.RegimenSection{}
	regimenSection2.RegimenName = "night"
	regimenSection2.RegimenSteps = []*common.DoctorInstructionItem{&common.DoctorInstructionItem{
		ParentId: regimenPlanRequest.AllRegimenSteps[1].Id,
		Text:     regimenPlanRequest.AllRegimenSteps[1].Text,
	},
	}
	regimenPlanRequest.RegimenSections = []*common.RegimenSection{regimenSection, regimenSection2}
	test_integration.CreateRegimenPlanForTreatmentPlan(regimenPlanRequest, testData, doctor, t)

	// submit the treatment plan
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	// now try to start a new treatment plan that is a version of the previous one
	tpResponse := test_integration.PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   treatmentPlan.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		ContentSourceType: common.TPContentSourceTypeTreatmentPlan,
		ContentSourceId:   treatmentPlan.Id,
	}, doctor, testData, t)

	if tpResponse.TreatmentPlan.Id.Int64() == treatmentPlan.Id.Int64() {
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
	} else if len(tpResponse.TreatmentPlan.Advice.SelectedAdvicePoints) != 2 {
		t.Fatalf("Expected 2 advice poitns instead got %d", len(tpResponse.TreatmentPlan.Advice.SelectedAdvicePoints))
	} else if tpResponse.TreatmentPlan.Advice.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the advice to be uncommitted but it wasnt")
	} else if len(tpResponse.TreatmentPlan.RegimenPlan.RegimenSections) != 2 {
		t.Fatalf("Expected 2 regimen sections instead got %d", len(tpResponse.TreatmentPlan.RegimenPlan.RegimenSections))
	} else if tpResponse.TreatmentPlan.RegimenPlan.Status != api.STATUS_UNCOMMITTED {
		t.Fatalf("Expected the regimen plan to be uncommitted but it wasnt")
	}

	// ensure that the content source is the treatment plan
	if tpResponse.TreatmentPlan.ContentSource == nil ||
		tpResponse.TreatmentPlan.ContentSource.ContentSourceType != common.TPContentSourceTypeTreatmentPlan {
		t.Fatalf("Expected the content source to be treatment plan but it wasnt")
	} else if tpResponse.TreatmentPlan.ContentSource.HasDeviated {
		t.Fatal("Didn't expect the treatment plan to deviate from the content source yet")
	}

	// now try to modify the treatment and it should mark the treatment plan as having deviated from the source
	treatment1.DispenseValue = encoding.HighPrecisionFloat64(21151)
	test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), tpResponse.TreatmentPlan.Id.Int64(), t)

	currentTreatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(tpResponse.TreatmentPlan.Id.Int64(), doctorId)
	if err != nil {
		t.Fatal(err)
	}

	if !currentTreatmentPlan.ContentSource.HasDeviated {
		t.Fatal("Expected the treatment plan to have deviated from the content source but it hasnt")
	}

}
