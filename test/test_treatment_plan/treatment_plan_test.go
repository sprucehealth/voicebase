package test_treatment_plan

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func jsonString(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestTreatmentPlanStatus(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	// get patient to start a visit
	_, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// this treatment plan should be in draft mode
	if !treatmentPlan.InDraftMode() {
		t.Fatalf("Expected treatmentPlan status to be in draft mode but it wasnt")
	}

	// once the doctor submits it it should become ACTIVE
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	drTreatmentPlan, err := testData.DataAPI.GetAbridgedTreatmentPlan(treatmentPlan.ID.Int64(), doctorID)
	test.OK(t, err)

	if drTreatmentPlan.Status != api.StatusActive {
		t.Fatalf("Expected status to be %s instead it was %s", api.StatusActive, drTreatmentPlan.Status)
	}
}

func TestTreatmentPlan_MarkPatientViewed(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.Doctor(dr.DoctorID, false)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	test.Equals(t, false, tp.PatientViewed)
	tp.PatientViewed = true
	test.OK(t, testData.DataAPI.UpdateTreatmentPlan(tp.ID.Int64(), &api.TreatmentPlanUpdate{
		PatientViewed: &tp.PatientViewed,
	}))
}

func TestTreatmentPlanList(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	// before submitting treatment plan if we try to get a list of treatment plans for patient there should be 1 in draft mode
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treamtent plan in draft instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plan in active mode instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now get a list of treatment plans for a patient
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)

	// there should be 1 active treatment plan for this patient
	if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 active treatment plan but got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	} else if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans in draft mode instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	}
}

func TestTreatmentPlanViews(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	dcli := test_integration.DoctorClient(testData, t, doctor.ID.Int64())
	pcli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	test_integration.AddTreatmentsToTreatmentPlan(tp.ID.Int64(), doctor, t, testData)
	_, guideIDs := test_integration.CreateTestResourceGuides(t, testData)
	dcli.AddResourceGuidesToTreatmentPlan(tp.ID.Int64(), guideIDs)
	test.OK(t, dcli.UpdateTreatmentPlanNote(tp.ID.Int64(), "foo"))
	test.OK(t, dcli.SubmitTreatmentPlan(tp.ID.Int64()))

	test.OK(t, testData.DataAPI.ActivateTreatmentPlan(tp.ID.Int64(), doctor.ID.Int64()))

	tpViews, err := pcli.TreatmentPlanForCase(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, false, tpViews == nil)
	test.Equals(t, 1, len(tpViews.HeaderViews))
	test.Equals(t, 3, len(tpViews.TreatmentViews))
	test.Equals(t, 2, len(tpViews.InstructionViews))
	exp := `{
  "type": "treatment:card_view",
  "views": [
    {
      "icon_url": "",
      "title": "Resources",
      "type": "treatment:card_title_view"
    },
    {
      "icon_height": 66,
      "icon_url": "http://example.com/blah.png",
      "icon_width": 66,
      "tap_url": "spruce:///action/view_resource_library_guide?guide_id=1",
      "text": "Guide 1",
      "type": "treatment:large_icon_text_button"
    },
    {
      "type": "treatment:small_divider"
    },
    {
      "icon_height": 66,
      "icon_url": "http://example.com/blah.png",
      "icon_width": 66,
      "tap_url": "spruce:///action/view_resource_library_guide?guide_id=2",
      "text": "Guide 2",
      "type": "treatment:large_icon_text_button"
    }
  ]
}`
	test.Equals(t, exp, jsonString(tpViews.InstructionViews[0]))
}

func TestTreatmentPlanList_DiffTPStates(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test_integration.SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	pcli := test_integration.PatientClient(testData, t, patient.PatientID.Int64())

	// in this submitted state the treatment plan should be visible to the doctor in the active list
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patient.PatientID.Int64(), doctor.AccountID.Int64(), testData, t)
	test.Equals(t, 1, len(treatmentPlanResponse.ActiveTreatmentPlans))
	test.Equals(t, tp.ID.Int64(), treatmentPlanResponse.ActiveTreatmentPlans[0].ID.Int64())

	// in this state the patient should not have an active treatment plan
	_, err = pcli.TreatmentPlanForCase(tp.PatientCaseID.Int64())
	test.Equals(t, false, err == nil)
	test.Equals(t, 404, err.(*apiservice.SpruceError).HTTPStatusCode)

	// now lets update the status of the treatment plan to put it in the rx_started state
	_, err = testData.DB.Exec(`update treatment_plan set status = ? where id = ?`, common.TPStatusRXStarted.String(), tp.ID.Int64())
	test.OK(t, err)

	// in this state the doctor should still be able to get the treatment plan as being in the active list
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patient.PatientID.Int64(), doctor.AccountID.Int64(), testData, t)
	test.Equals(t, 1, len(treatmentPlanResponse.ActiveTreatmentPlans))
	test.Equals(t, tp.ID.Int64(), treatmentPlanResponse.ActiveTreatmentPlans[0].ID.Int64())

	// in this state the patient should not have an active treatment plan
	_, err = pcli.TreatmentPlanForCase(tp.PatientCaseID.Int64())
	test.Equals(t, false, err == nil)
	test.Equals(t, 404, err.(*apiservice.SpruceError).HTTPStatusCode)

	// now lets activate the treatment plan
	err = testData.DataAPI.ActivateTreatmentPlan(tp.ID.Int64(), doctor.ID.Int64())
	test.OK(t, err)

	// in this state the patient should have an active treatment plan
	tpViews, err := pcli.TreatmentPlanForCase(tp.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, false, tpViews == nil)
}

func TestTreatmentPlanList_DraftTest(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)

	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	doctor2, err := testData.DataAPI.GetDoctorFromID(signedUpDoctorResponse.DoctorID)
	test.OK(t, err)

	// add doctor2 to the care team of the patient
	test.OK(t, testData.DataAPI.AddDoctorToPatientCase(doctor2.ID.Int64(), treatmentPlan.PatientCaseID.Int64()))

	// doctor2 should not be able to see previous doctor's draft
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor2.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// now doctor2 should be able to see the treatment plan that doctor1 just submitted
	treatmentPlanResponse = test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor2.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanList_FavTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	favoriteTreatmentPlan := test_integration.CreateFavoriteTreatmentPlan(treatmentPlan.ID.Int64(), testData, doctor, t)
	responseData := test_integration.PickATreatmentPlanForPatientVisit(patientVisitResponse.PatientVisitID, doctor, favoriteTreatmentPlan, testData, t)

	// favorite treatment plan information should be included in the list of treatment plans
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}

	// now lets attempt to get this treatment plan by id to ensure that its linked to favorite treatment plan
	drTreatmentPlan := test_integration.GetDoctorTreatmentPlanByID(treatmentPlanResponse.DraftTreatmentPlans[0].ID.Int64(), doctor.AccountID.Int64(), testData, t)
	if drTreatmentPlan.ContentSource == nil || drTreatmentPlan.ContentSource.ID.Int64() == 0 {
		t.Fatalf("Expected link to favorite treatment plan to exist but it doesnt")
	} else if drTreatmentPlan.ContentSource.ID.Int64() != favoriteTreatmentPlan.ID.Int64() {
		t.Fatalf("Expected treatment plan to be linked to fav treatment plan id %d but instead it ewas linked to id %d", favoriteTreatmentPlan.ID.Int64(), drTreatmentPlan.ContentSource.ID.Int64())
	}

	// lets submit the treatment plan back to patient so that we can test whether or not favorite tretment plan information is shown to another doctor
	// it shouldn't be
	test_integration.SubmitPatientVisitBackToPatient(responseData.TreatmentPlan.ID.Int64(), doctor, testData, t)

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(signedUpDoctorResponse.DoctorID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	patientCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	// assign the doctor to the patient case
	test.OK(t, testData.DataAPI.AddDoctorToPatientCase(doctor2.ID.Int64(), patientCase.ID.Int64()))

	drTreatmentPlan = test_integration.GetDoctorTreatmentPlanByID(treatmentPlanResponse.DraftTreatmentPlans[0].ID.Int64(), doctor2.AccountID.Int64(), testData, t)
	if drTreatmentPlan.ContentSource != nil && drTreatmentPlan.ContentSource.ID.Int64() != 0 {
		t.Fatalf("Expected content source to indicate that treatment plan deviated from original content source but it doesnt")
	}
}

func TestTreatmentPlanDelete(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	// should be able to delete this treatment plan owned by doctor
	test.OK(t, cli.DeleteTreatmentPlan(treatmentPlan.ID.Int64()))

	// there should be no drafts left given that we just deleted it
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanDelete_ActiveTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctorID)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	// submit treatment plan to patient to make it active
	test_integration.SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	// attempting to delete the treatment plan should fail given that the treatment plan is active
	if err := cli.DeleteTreatmentPlan(treatmentPlan.ID.Int64()); err == nil {
		t.Fatal("Expected a BadRequest error but got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected a SpruceError. Got %T: %s", err, err.Error())
	} else if e.HTTPStatusCode != http.StatusBadRequest {
		t.Fatalf("Expectes status BadRequest got %d", e.HTTPStatusCode)
	}

	// there should still exist an active treatment plan
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

func TestTreatmentPlanDelete_DifferentDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	patientVisitResponse, treatmentPlan := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patientID, err := testData.DataAPI.GetPatientIDFromPatientVisitID(patientVisitResponse.PatientVisitID)
	test.OK(t, err)

	signedUpDoctorResponse, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(signedUpDoctorResponse.DoctorID)
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctor2.ID.Int64())

	// attempting to delete the treatment plan should fail given that the treatment plan is being worked on by another doctor
	if err := cli.DeleteTreatmentPlan(treatmentPlan.ID.Int64()); err == nil {
		t.Fatal("Expected a Forbidden error but got no error")
	} else if e, ok := err.(*apiservice.SpruceError); !ok {
		t.Fatalf("Expected a SpruceError. Got %T: %s", err, err.Error())
	} else if e.HTTPStatusCode != http.StatusForbidden {
		t.Fatalf("Expectes status Forbidden got %d", e.HTTPStatusCode)
	}

	// there should still exist an active treatment plan
	treatmentPlanResponse := test_integration.GetListOfTreatmentPlansForPatient(patientID, doctor.AccountID.Int64(), testData, t)
	if len(treatmentPlanResponse.DraftTreatmentPlans) != 1 {
		t.Fatalf("Expected 1 treatment plan instead got %d", len(treatmentPlanResponse.DraftTreatmentPlans))
	} else if len(treatmentPlanResponse.ActiveTreatmentPlans) != 0 {
		t.Fatalf("Expected no treatment plans instead got %d", len(treatmentPlanResponse.ActiveTreatmentPlans))
	}
}

// This test is to ensure that draft and active TPs are being queried for separately for each case
func TestTreatmentPlan_MultipleCases(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	_, tp1 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	p := test_integration.CreatePathway(t, testData, "pathway2")
	patient, err := testData.DataAPI.GetPatientFromID(tp1.PatientID)
	test.OK(t, err)

	_, tp2 := test_integration.CreateRandomPatientVisitAndPickTPForPathway(t, testData, p, patient, doctor)

	// now make a call to get the case list and ensure that each has a different draft tp listed
	dc := test_integration.DoctorClient(testData, t, dr.DoctorID)
	cases, err := dc.CasesForPatient(patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	test.Equals(t, 1, len(cases[0].DraftTPs))
	test.Equals(t, 1, len(cases[1].DraftTPs))
	test.Equals(t, true, cases[0].DraftTPs[0].ID != cases[1].DraftTPs[0].ID)
	test.Equals(t, cases[0].ID, cases[0].DraftTPs[0].PatientCaseID.Int64())
	test.Equals(t, cases[1].ID, cases[1].DraftTPs[0].PatientCaseID.Int64())

	// lets submit each case
	test_integration.SubmitPatientVisitBackToPatient(tp1.ID.Int64(), doctor, testData, t)
	test_integration.SubmitPatientVisitBackToPatient(tp2.ID.Int64(), doctor, testData, t)

	// now ensure that these TPs come up as active TPs
	cases, err = dc.CasesForPatient(patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 0, len(cases[0].DraftTPs))
	test.Equals(t, 0, len(cases[1].DraftTPs))
	test.Equals(t, 1, len(cases[0].ActiveTPs))
	test.Equals(t, 1, len(cases[1].ActiveTPs))
	test.Equals(t, true, cases[0].ActiveTPs[0].ID != cases[1].ActiveTPs[0].ID)
	test.Equals(t, cases[0].ID, cases[0].ActiveTPs[0].PatientCaseID.Int64())
	test.Equals(t, cases[1].ID, cases[1].ActiveTPs[0].PatientCaseID.Int64())
}

func TestTreatmentPlanSections(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	doctorID := test_integration.GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	cli := test_integration.DoctorClient(testData, t, doctorID)

	visit, tp0 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	test.OK(t, cli.UpdateTreatmentPlanNote(tp0.ID.Int64(), "Some note"))
	test_integration.AddTreatmentsToTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)
	test_integration.AddRegimenPlanForTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)

	tp, err := cli.TreatmentPlan(tp0.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.Equals(t, false, tp.RegimenPlan == nil)
	test.Equals(t, false, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.NoSections)
	test.OK(t, err)
	test.Equals(t, true, tp.RegimenPlan == nil)
	test.Equals(t, true, tp.TreatmentList == nil)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.TreatmentsSection)
	test.OK(t, err)
	test.Equals(t, true, tp.RegimenPlan == nil)
	test.Equals(t, false, tp.TreatmentList == nil)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.RegimenSection|doctor_treatment_plan.NoteSection)
	test.OK(t, err)
	test.Equals(t, false, tp.RegimenPlan == nil)
	test.Equals(t, true, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	// Make sure a TP created from an FTP (derived source) also works

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.OK(t, cli.DeleteTreatmentPlan(tp.ID.Int64()))

	ftp := &responses.FavoriteTreatmentPlan{
		Name:          "Test FTP",
		RegimenPlan:   tp.RegimenPlan,
		TreatmentList: tp.TreatmentList,
		Note:          tp.Note,
	}
	ftp, err = cli.CreateFavoriteTreatmentPlan(ftp)
	test.OK(t, err)

	tp, err = cli.PickTreatmentPlanForVisit(visit.PatientVisitID, ftp)
	test.OK(t, err)
	test.Equals(t, false, tp.RegimenPlan == nil)
	test.Equals(t, false, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)
	test.Equals(t, false, tp.RegimenPlan == nil)
	test.Equals(t, false, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.NoSections)
	test.OK(t, err)
	test.Equals(t, true, tp.RegimenPlan == nil)
	test.Equals(t, true, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.TreatmentsSection)
	test.OK(t, err)
	test.Equals(t, true, tp.RegimenPlan == nil)
	test.Equals(t, false, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)

	tp, err = cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.RegimenSection|doctor_treatment_plan.NoteSection)
	test.OK(t, err)
	test.Equals(t, false, tp.RegimenPlan == nil)
	test.Equals(t, true, tp.TreatmentList == nil)
	test.Equals(t, "Some note", tp.Note)
}

// This test is to ensure that draft and active TPs are being queried for separately for each case
func TestTreatmentPlan_GlobalTreatmentPlanAddition(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr3, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	memberships1, err := testData.DataAPI.FTPMembershipsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships1))
	memberships2, err := testData.DataAPI.FTPMembershipsForDoctor(dr2.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships2))
	memberships3, err := testData.DataAPI.FTPMembershipsForDoctor(dr3.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(memberships3))

	p1 := &common.Pathway{
		Tag:            "p1",
		Name:           "p1",
		MedicineBranch: "p1",
		Status:         "ACTIVE",
	}
	p2 := &common.Pathway{
		Tag:            "p2",
		Name:           "p2",
		MedicineBranch: "p2",
		Status:         "ACTIVE",
	}
	testData.DataAPI.CreatePathway(p1)
	testData.DataAPI.CreatePathway(p2)
	p1, err = testData.DataAPI.PathwayForTag("p1", api.PONone)
	test.OK(t, err)
	p2, err = testData.DataAPI.PathwayForTag("p2", api.PONone)
	test.OK(t, err)
	ftp1 := &common.FavoriteTreatmentPlan{
		Name:      "ftp1",
		Note:      "ftp1",
		Lifecycle: "ACTIVE",
	}
	ftp2 := &common.FavoriteTreatmentPlan{
		Name:      "ftp2",
		Note:      "ftp2",
		Lifecycle: "ACTIVE",
	}
	ftpsByPathwayID := map[int64][]*common.FavoriteTreatmentPlan{
		p1.ID: []*common.FavoriteTreatmentPlan{ftp1},
		p2.ID: []*common.FavoriteTreatmentPlan{ftp2},
	}
	err = testData.DataAPI.InsertGlobalFTPsAndUpdateMemberships(ftpsByPathwayID)
	test.OK(t, err)
	memberships1, err = testData.DataAPI.FTPMembershipsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(memberships1))
	memberships2, err = testData.DataAPI.FTPMembershipsForDoctor(dr2.DoctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(memberships2))
	memberships3, err = testData.DataAPI.FTPMembershipsForDoctor(dr3.DoctorID)
	test.OK(t, err)
	test.Equals(t, 2, len(memberships3))
	notIn := []int64{memberships1[0].DoctorFavoritePlanID, memberships1[1].DoctorFavoritePlanID}
	ftp1 = &common.FavoriteTreatmentPlan{
		Name:      "ftp1",
		Note:      "ftp1",
		Lifecycle: "ACTIVE",
	}
	ftp2 = &common.FavoriteTreatmentPlan{
		Name:      "ftp2",
		Note:      "ftp2",
		Lifecycle: "ACTIVE",
	}
	ftp3 := &common.FavoriteTreatmentPlan{
		Name:      "ftp3",
		Note:      "ftp3",
		Lifecycle: "ACTIVE",
	}
	ftp4 := &common.FavoriteTreatmentPlan{
		Name:      "ftp4",
		Note:      "ftp4",
		Lifecycle: "ACTIVE",
	}
	ftpsByPathwayID = map[int64][]*common.FavoriteTreatmentPlan{
		p1.ID: []*common.FavoriteTreatmentPlan{ftp1, ftp2, ftp4},
		p2.ID: []*common.FavoriteTreatmentPlan{ftp3},
	}
	err = testData.DataAPI.InsertGlobalFTPsAndUpdateMemberships(ftpsByPathwayID)
	test.OK(t, err)
	memberships1, err = testData.DataAPI.FTPMembershipsForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 4, len(memberships1))
	memberships2, err = testData.DataAPI.FTPMembershipsForDoctor(dr2.DoctorID)
	test.OK(t, err)
	test.Equals(t, 4, len(memberships2))
	memberships3, err = testData.DataAPI.FTPMembershipsForDoctor(dr3.DoctorID)
	test.OK(t, err)
	test.Equals(t, 4, len(memberships3))
	for _, v := range memberships1 {
		for _, v2 := range notIn {
			test.Assert(t, v.DoctorFavoritePlanID != v2, "Expected %d to not be present in membership set.", v)
		}
	}
	for _, v := range memberships2 {
		for _, v2 := range notIn {
			test.Assert(t, v.DoctorFavoritePlanID != v2, "Expected %d to not be present in membership set.", v)
		}
	}
	for _, v := range memberships3 {
		for _, v2 := range notIn {
			test.Assert(t, v.DoctorFavoritePlanID != v2, "Expected %d to not be present in membership set.", v)
		}
	}
}
