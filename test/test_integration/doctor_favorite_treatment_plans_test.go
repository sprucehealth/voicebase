package test_integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

func TestFavoriteTreatmentPlan(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}

	cli := DoctorClient(testData, t, doctorID)

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	originalRegimenPlan := favoriteTreatmentPlan.RegimenPlan

	// now lets go ahead and update the favorite treatment plan

	updatedName := "Updating name"
	favoriteTreatmentPlan.Name = updatedName
	favoriteTreatmentPlan.RegimenPlan.Sections = favoriteTreatmentPlan.RegimenPlan.Sections[1:]

	if ftp, err := cli.UpdateFavoriteTreatmentPlan(favoriteTreatmentPlan); err != nil {
		t.Fatal(err)
	} else if len(ftp.RegimenPlan.Sections) != 1 {
		t.Fatalf("Expected 1 section in the regimen plan instead got %d", len(ftp.RegimenPlan.Sections))
	} else if ftp.Name != updatedName {
		t.Fatalf("Expected name of favorite treatment plan to be %s instead got %s", updatedName, ftp.Name)
	}

	// lets go ahead and add another favorited treatment
	favoriteTreatmentPlan2 := &responses.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan #2",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{{
				DrugDBIDs: map[string]string{
					erx.LexiDrugSynID:     "1234",
					erx.LexiGenProductID:  "12345",
					erx.LexiSynonymTypeID: "123556",
					erx.NDC:               "2415",
				},
				DrugInternalName:        "Drug1 (Route1 - Form1)",
				DosageStrength:          "Strength1",
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
		RegimenPlan: originalRegimenPlan,
	}

	if _, err := cli.CreateFavoriteTreatmentPlan(favoriteTreatmentPlan2); err != nil {
		t.Fatal(err)
	}

	ftps, err := cli.ListFavoriteTreatmentPlansForTag(api.AcnePathwayTag)
	fmt.Println(ftps)
	if err != nil {
		t.Fatal(err)
	} else if len(ftps) != 2 {
		t.Fatalf("Expected 2 favorite treatment plans instead got %d", len(ftps))
	} else if len(ftps[0].RegimenPlan.Sections) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 1 regimen section")
	} else if len(ftps[1].RegimenPlan.Sections) != 1 {
		t.Fatalf("Expected favorite treatment plan to have 2 regimen sections")
	}

	// lets go ahead and delete favorite treatment plan
	if err := cli.DeleteFavoriteTreatmentPlan(ftps[0].ID.Int64()); err != nil {
		t.Fatal(err)
	}
}

func TestFTP_MultiplePathways(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	dc := DoctorClient(testData, t, doctorID)

	// create test pathway
	pathway := &common.Pathway{
		Name:           "test",
		Tag:            "test",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	// create the test SKU
	sku := &common.SKU{
		Type:         "test_visit",
		CategoryType: common.SCVisit,
	}
	_, err = testData.DataAPI.CreateSKU(sku)
	test.OK(t, err)

	pr := SignupRandomTestPatient(t, testData)
	patient := pr.Patient

	_, treatmentPlan := CreateRandomPatientVisitAndPickTPForPathway(t, testData, pathway, patient, doctor)

	// create the regimen plan and treatments for the treatment plan
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: treatmentPlan.ID,
		Sections: []*common.RegimenSection{
			{
				Name: "morning",
				Steps: []*common.DoctorInstructionItem{
					{
						Text: "step 1",
					},
					{
						Text: "step 2",
					},
				},
			},
			{
				Name: "night",
				Steps: []*common.DoctorInstructionItem{{
					Text: "step 2",
				}},
			},
		},
	}

	regimenPlanResponse, err := dc.CreateRegimenPlan(regimenPlanRequest)
	test.OK(t, err)

	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		DosageStrength:          "Strength1",
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
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// lets add a favorite treatment plan for doctor. This should get grouped against the pathway associatd with the case.
	favoriteTreatmentPlan := &responses.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: regimenPlanResponse.AllSteps,
			Sections: regimenPlanResponse.Sections,
		},
	}

	ftp, err := dc.CreateFavoriteTreatmentPlanFromTreatmentPlan(favoriteTreatmentPlan, treatmentPlan.ID.Int64())
	test.OK(t, err)

	// lets attempt to get this ftp
	ftpsForPathway1, err := dc.ListFavoriteTreatmentPlansForTag(pathway.Tag)
	test.OK(t, err)
	test.Equals(t, 1, len(ftpsForPathway1))
	test.Equals(t, ftp.ID.Int64(), ftpsForPathway1[0].ID.Int64())

	// lets ensure that this FTP is not pulled against the acne pathway
	ftpsForAcnePathway, err := dc.ListFavoriteTreatmentPlansForTag(api.AcnePathwayTag)
	test.OK(t, err)
	test.Equals(t, 0, len(ftpsForAcnePathway))

}

// This test ensures to check that after deleting a FTP, the TP that was created
// from the FTP has its content source deleted and getting the TP still works
func TestFavoriteTreatmentPlan_DeletingFTP(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	// lets start a new TP based on FTP
	responseData := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypePatientVisit,
		ParentID:   encoding.NewObjectID(patientVisitResponse.PatientVisitID),
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   favoriteTreatmentPlan.ID,
	}, doctor, testData, t)

	// ensure that this TP has the FTP as its content source
	if responseData.TreatmentPlan.ContentSource == nil ||
		responseData.TreatmentPlan.ContentSource.Type != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ID != favoriteTreatmentPlan.ID.Int64() {
		t.Fatal("Expected the newly created Treatment plan to have the FTP as its source")
	}

	// now lets go ahead and delete the FTP
	if err := cli.DeleteFavoriteTreatmentPlan(favoriteTreatmentPlan.ID.Int64()); err != nil {
		t.Fatal(err)
	}

	// now if we try to get the TP initially created from the FTP, the content source should not exist
	if tp, err := cli.TreatmentPlan(responseData.TreatmentPlan.ID.Int64(), false, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource != nil {
		t.Fatal("Expected nil content source for treatment plan after deleting FTP from which the TP was started")
	}
}

// This test ensures that even if an FTP is deleted that was picked as content source for a TP that has been activated for a patient,
// the content source gets deleted while TP remains unaltered
func TestFavoriteTreatmentPlan_DeletingFTP_ActiveTP(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	favoriteTreatmentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	// lets start a new TP based on FTP
	responseData := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypePatientVisit,
		ParentID:   encoding.NewObjectID(patientVisitResponse.PatientVisitID),
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   favoriteTreatmentPlan.ID,
	}, doctor, testData, t)

	// ensure that this TP has the FTP as its content source
	if responseData.TreatmentPlan.ContentSource == nil ||
		responseData.TreatmentPlan.ContentSource.Type != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ID != favoriteTreatmentPlan.ID.Int64() {
		t.Fatal("Expected the newly created Treatment plan to have the FTP as its source")
	}

	// submit the treatments for the TP
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreatmentPlan.TreatmentList.Treatments, doctor.AccountID.Int64(), responseData.TreatmentPlan.ID.Int64(), t)

	// submit regimen for TP
	regimenPlan := &common.RegimenPlan{
		TreatmentPlanID: responseData.TreatmentPlan.ID,
		Sections:        favoriteTreatmentPlan.RegimenPlan.Sections,
	}
	if _, err := cli.CreateRegimenPlan(regimenPlan); err != nil {
		t.Fatal(err)
	}

	test.OK(t, cli.UpdateTreatmentPlanNote(responseData.TreatmentPlan.ID.Int64(), favoriteTreatmentPlan.Note))
	test.OK(t, cli.SubmitTreatmentPlan(responseData.TreatmentPlan.ID.Int64()))
	test.OK(t, cli.DeleteFavoriteTreatmentPlan(favoriteTreatmentPlan.ID.Int64()))

	// now if we try to get the TP initially created from the FTP, the content source should not exist
	if tp, err := cli.TreatmentPlan(responseData.TreatmentPlan.ID.Int64(), false, doctor_treatment_plan.AllSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource != nil {
		t.Fatal("Expected nil content source for treatment plan after deleting FTP from which the TP was started")
	} else if !tp.IsActive() {
		t.Fatalf("Expected the treatment plan to be active but it wasnt")
	} else if !favoriteTreatmentPlan.EqualsTreatmentPlan(tp) {
		t.Fatal("Even though the FTP was deleted, the contents of the TP and FTP should still match")
	}
}

func TestFavoriteTreatmentPlan_PickingAFavoriteTreatmentPlan(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	if tp, err := cli.TreatmentPlan(treatmentPlan.ID.Int64(), false, doctor_treatment_plan.TreatmentsSection|doctor_treatment_plan.RegimenSection); err != nil {
		t.Fatal(err)
	} else if tp.TreatmentList != nil && len(tp.TreatmentList.Treatments) != 0 {
		t.Fatalf("Expected there to exist no treatments in treatment plan")
	} else if tp.RegimenPlan != nil && len(tp.RegimenPlan.Sections) != 0 {
		t.Fatalf("Expected regimen to not exist for treatment plan instead we have %d regimen sections", len(tp.RegimenPlan.Sections))
	} else if len(tp.RegimenPlan.AllSteps) == 0 {
		t.Fatalf("Expected regimen steps to exist given that they were created to create the treatment plan")
	}

	// now lets attempt to pick the added favorite treatment plan and compare the two again
	// this time the treatment plan should be populated with data from the favorite treatment plan
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitID, doctor, favoriteTreamentPlan, testData, t)
	if responseData.TreatmentPlan == nil {
		t.Fatalf("Expected treatment plan to exist")
	} else if responseData.TreatmentPlan.TreatmentList != nil && len(responseData.TreatmentPlan.TreatmentList.Treatments) != 1 {
		t.Fatalf("Expected there to exist no treatments in treatment plan")
	} else if responseData.TreatmentPlan.TreatmentList.Status != api.StatusUncommitted {
		t.Fatalf("Status should indicate UNCOMMITTED for treatment section when the doctor has not committed the section")
	} else if responseData.TreatmentPlan.RegimenPlan != nil && len(responseData.TreatmentPlan.RegimenPlan.Sections) != 2 {
		t.Fatalf("Expected regimen to not exist for treatment plan instead we have %d regimen sections", len(responseData.TreatmentPlan.RegimenPlan.Sections))
	} else if len(responseData.TreatmentPlan.RegimenPlan.AllSteps) != 2 {
		t.Fatalf("Expected there to exist 2 regimen steps in the master list instead got %d", len(responseData.TreatmentPlan.RegimenPlan.AllSteps))
	} else if responseData.TreatmentPlan.RegimenPlan.Status != api.StatusUncommitted {
		t.Fatalf("Status should indicate UNCOMMITTED for regimen plan when the doctor has not committed the section")
	} else if !favoriteTreamentPlan.EqualsTreatmentPlan(responseData.TreatmentPlan) {
		t.Fatal("Expected the contents of the favorite treatment plan to be the same as that of the treatment plan but its not")
	}

	var count int64
	if err := testData.DB.QueryRow(`select count(*) from treatment_plan inner join treatment_plan_patient_visit_mapping on treatment_plan_id = treatment_plan.id where patient_visit_id = ?`, patientVisitResponse.PatientVisitID).Scan(&count); err != nil {
		t.Fatalf("Unable to query database to get number of treatment plans for patient visit: %s", err)
	} else if count != 1 {
		t.Fatalf("Expected 1 treatment plan for patient visit instead got %d", count)
	}
}

func TestFavoriteTreatmentPlan_BreakingMappingOnModify(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	// pick this favorite treatment plan for the visit
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitID, doctor, favoriteTreamentPlan, testData, t)

	// lets attempt to modify and submit regimen section for patient visit
	regimenPlanRequest := &common.RegimenPlan{
		TreatmentPlanID: responseData.TreatmentPlan.ID,
		AllSteps:        favoriteTreamentPlan.RegimenPlan.AllSteps,
		Sections:        favoriteTreamentPlan.RegimenPlan.Sections[:1],
	}
	if _, err := cli.CreateRegimenPlan(regimenPlanRequest); err != nil {
		t.Fatal(err)
	}

	// the regimen plan should indicate that it was committed while the rest of the sections
	// should continue to be in the UNCOMMITTED state
	if tp, err := cli.TreatmentPlan(responseData.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil || tp.ContentSource.Type != common.TPContentSourceTypeFTP ||
		tp.ContentSource.ID == 0 || !tp.ContentSource.Deviated {
		t.Fatalf("Expected the treatment plan to indicate that it has deviated from the original content source (ftp) but it doesnt do so")
	}

	// lets try modfying treatments on a new treatment plan picked from favorites
	responseData = PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitID, doctor, favoriteTreamentPlan, testData, t)

	// lets make sure linkage exists
	if responseData.TreatmentPlan.ContentSource == nil || responseData.TreatmentPlan.ContentSource.Type != common.TPContentSourceTypeFTP ||
		responseData.TreatmentPlan.ContentSource.ID == 0 {
		t.Fatalf("Expected the treatment plan to come from a favorite treatment plan")
	} else if responseData.TreatmentPlan.ContentSource.ID != favoriteTreamentPlan.ID.Int64() {
		t.Fatalf("Got a different favorite treatment plan linking to the treatment plan. Expected %v got %v", favoriteTreamentPlan.ID, responseData.TreatmentPlan.ContentSource.ID)
	}

	// modify treatment
	favoriteTreamentPlan.TreatmentList.Treatments[0].DispenseValue = encoding.HighPrecisionFloat64(123.12345)
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreamentPlan.TreatmentList.Treatments, doctor.AccountID.Int64(), responseData.TreatmentPlan.ID.Int64(), t)

	// linkage should now be broken
	if tp, err := cli.TreatmentPlan(responseData.TreatmentPlan.ID.Int64(), false, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil || tp.ContentSource.Type != common.TPContentSourceTypeFTP ||
		tp.ContentSource.ID == 0 || !tp.ContentSource.Deviated {
		t.Fatalf("Expected the treatment plan to indicate that it has deviated from the original content source (ftp) but it doesnt do so")
	}

}

// This test is to cover the scenario where if a doctor modifies,say, the treatment section after
// starting from a favorite treatment plan, we ensure that the rest of the sections are still prefilled
// with the contents of the favorite treatment plan
func TestFavoriteTreatmentPlan_BreakingMappingOnModify_PrefillRestOfData(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// create a favorite treatment plan
	favoriteTreamentPlan := CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)

	// pick this favorite treatment plan for the visit
	responseData := PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitID, doctor, favoriteTreamentPlan, testData, t)

	// modify treatment
	favoriteTreamentPlan.TreatmentList.Treatments[0].DispenseValue = encoding.HighPrecisionFloat64(123.12345)
	AddAndGetTreatmentsForPatientVisit(testData, favoriteTreamentPlan.TreatmentList.Treatments, doctor.AccountID.Int64(), responseData.TreatmentPlan.ID.Int64(), t)

	if tp, err := cli.TreatmentPlan(responseData.TreatmentPlan.ID.Int64(), false, doctor_treatment_plan.RegimenSection|doctor_treatment_plan.TreatmentsSection); err != nil {
		t.Fatal(err)
	} else if tp.TreatmentList == nil || len(tp.TreatmentList.Treatments) == 0 {
		t.Fatal("Expected treatments to exist")
	} else if tp.RegimenPlan == nil || len(tp.RegimenPlan.Sections) == 0 || tp.RegimenPlan.Status != api.StatusUncommitted {
		t.Fatal("Expected regimen plan to be prefilled with FTP and be in UNCOMMITTED state")
	}
}

// This test ensures that the user can create a favorite treatment plan
// in the context of treatment plan by specifying the treatment plan to associate the
// favorite treatment plan with
func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
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
		Steps: []*common.DoctorInstructionItem{
			{
				Text:  regimenStep1.Text,
				State: common.StateAdded,
			},
			{
				Text:  regimenStep2.Text,
				State: common.StateAdded,
			},
		},
	}

	regimenSection2 := &common.RegimenSection{
		Name: "night",
		Steps: []*common.DoctorInstructionItem{{
			Text:  regimenStep2.Text,
			State: common.StateAdded,
		}},
	}

	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanRequest.Sections = []*common.RegimenSection{regimenSection, regimenSection2}
	regimenPlanResponse, err := cli.CreateRegimenPlan(regimenPlanRequest)
	test.OK(t, err)

	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	// prepare the regimen steps and the advice points to be added into the sections
	// after the global list for each has been updated to include items.
	// the reason this is important is because favorite treatment plans require items to exist that are linked
	// from the master list
	regimenSection.Steps[0].ParentID = regimenPlanResponse.AllSteps[0].ID
	regimenSection.Steps[1].ParentID = regimenPlanResponse.AllSteps[1].ID
	regimenSection2.Steps[0].ParentID = regimenPlanResponse.AllSteps[1].ID

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		DosageStrength:          "Strength1",
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
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &responses.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: regimenPlanResponse.AllSteps,
			Sections: []*common.RegimenSection{regimenSection, regimenSection2},
		},
	}

	ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(favoriteTreatmentPlan, treatmentPlan.ID.Int64())
	test.OK(t, err)
	if ftp.RegimenPlan == nil || len(ftp.RegimenPlan.Sections) != 2 {
		t.Fatalf("Expected to have a regimen plan or 2 items in the regimen section")
	}

	abbreviatedTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(treatmentPlan.ID.Int64(), doctorID)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource == nil || abbreviatedTreatmentPlan.ContentSource.Type != common.TPContentSourceTypeFTP ||
		abbreviatedTreatmentPlan.ContentSource.ID.Int64() != ftp.ID.Int64() {
		t.Fatalf("Expected the link between treatmenet plan and favorite treatment plan to exist but it doesnt")
	}
}

func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan_EmptyRegimen(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
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
	regimenPlanResponse, err := cli.CreateRegimenPlan(regimenPlanRequest)
	test.OK(t, err)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		DosageStrength:          "Strength1",
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
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &responses.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: regimenPlanResponse.AllSteps,
		},
	}

	ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(favoriteTreatmentPlan, treatmentPlan.ID.Int64())
	if err != nil {
		t.Fatal(err)
	}

	abbreviatedTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(treatmentPlan.ID.Int64(), doctorID)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource == nil || abbreviatedTreatmentPlan.ContentSource.Type != common.TPContentSourceTypeFTP ||
		abbreviatedTreatmentPlan.ContentSource.ID.Int64() != ftp.ID.Int64() {
		t.Fatalf("Expected the link between treatmenet plan and favorite treatment plan to exist but it doesnt")
	}

}

func TestFavoriteTreatmentPlan_InContextOfTreatmentPlan_TwoDontMatch(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	if err != nil {
		t.Fatalf("Unable to get doctor from id: %s", err)
	}
	cli := DoctorClient(testData, t, doctorID)

	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
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

	regimenPlanRequest.Sections = []*common.RegimenSection{
		&common.RegimenSection{
			Name: "dgag",
			Steps: []*common.DoctorInstructionItem{
				regimenStep1,
			},
		},
	}
	regimenPlanRequest.AllSteps = []*common.DoctorInstructionItem{regimenStep1, regimenStep2}
	regimenPlanResponse, err := cli.CreateRegimenPlan(regimenPlanRequest)
	test.OK(t, err)
	ValidateRegimenRequestAgainstResponse(regimenPlanRequest, regimenPlanResponse, t)

	treatment1 := &common.Treatment{
		DrugDBIDs: map[string]string{
			erx.LexiDrugSynID:     "1234",
			erx.LexiGenProductID:  "12345",
			erx.LexiSynonymTypeID: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Drug1 (Route1 - Form1)",
		DosageStrength:          "Strength1",
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
	}

	AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountID.Int64(), treatmentPlan.ID.Int64(), t)

	// lets add a favorite treatment plan for doctor
	favoriteTreatmentPlan := &responses.FavoriteTreatmentPlan{
		Name: "Test Treatment Plan",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: regimenPlanResponse.AllSteps,
		},
	}
	if _, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(favoriteTreatmentPlan, treatmentPlan.ID.Int64()); err == nil {
		t.Fatal("Expected BadRequest got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected a SpruceError. Got %T: %s", err, err.Error())
	} else if e.HTTPStatusCode != http.StatusBadRequest {
		t.Fatalf("Expectes status BadRequest got %d", e.HTTPStatusCode)
	}

	abbreviatedTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(treatmentPlan.ID.Int64(), doctorID)
	if err != nil {
		t.Fatalf("Unable to get abbreviated favorite treatment plan: %s", err)
	} else if abbreviatedTreatmentPlan.ContentSource != nil {
		t.Fatalf("Expected the treatment plan to not indicate that it was linked to another doctor's favorite treatment plan")
	}

}
