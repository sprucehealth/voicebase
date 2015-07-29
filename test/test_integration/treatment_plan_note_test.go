package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/test"
)

func TestTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dres.DoctorID)
	test.OK(t, err)

	cli := DoctorClient(testData, t, dres.DoctorID)

	// Create a patient treatment plan, and save a draft message
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), note); err != nil {
		t.Fatal(err)
	}
	if tp, err := cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.NoteSection); err != nil {
		t.Fatal(err)
	} else if tp.Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, tp.Note)
	}

	// Update treatment plan message
	note = "Dear foo, I have changed my mind"
	if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), note); err != nil {
		t.Fatal(err)
	}

	if tp, err := cli.TreatmentPlan(tp.ID.Int64(), false, doctor_treatment_plan.NoteSection); err != nil {
		t.Fatal(err)
	} else if tp.Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, tp.Note)
	}
}

func TestVersionedTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dres.DoctorID)
	test.OK(t, err)

	cli := DoctorClient(testData, t, dres.DoctorID)

	// Create a patient treatment plan and set the note
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), note); err != nil {
		t.Fatal(err)
	}
	ftp := CreateFavoriteTreatmentPlan(tp.ID.Int64(), testData, doctor, t)

	SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// Started from scratch
	tpNew := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != "" {
		t.Fatalf("Expected empty note got '%s'", tpNew.TreatmentPlan.Note)
	}

	// Started from a treatment plan
	tpNew = PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeTreatmentPlan,
		ID:   tp.ID,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != doctor_treatment_plan.VersionedTreatmentPlanNote {
		t.Fatalf("Expected '%s' got '%s'", doctor_treatment_plan.VersionedTreatmentPlanNote, tpNew.TreatmentPlan.Note)
	}

	// Started from a favorite treatment plan
	tpNew = PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   ftp.ID,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != ftp.Note {
		t.Fatalf("Expected '%s' got '%s'", ftp.Note, tpNew.TreatmentPlan.Note)
	}

	// Make sure note is maintained after submitting
	SubmitPatientVisitBackToPatient(tpNew.TreatmentPlan.ID.Int64(), doctor, testData, t)
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), false, doctor_treatment_plan.NoteSection); err != nil {
		t.Fatal(err)
	} else if ftp.Note != tpNew.TreatmentPlan.Note {
		t.Fatalf("Expected '%s' got '%s'", note, tp.Note)
	}
}

func TestTreatmentPlanNoteDeviation(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dres.DoctorID)
	test.OK(t, err)

	cli := DoctorClient(testData, t, dres.DoctorID)

	// Create a patient treatment plan and set the note
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), note); err != nil {
		t.Fatal(err)
	}
	ftp := CreateFavoriteTreatmentPlan(tp.ID.Int64(), testData, doctor, t)
	SubmitPatientVisitBackToPatient(tp.ID.Int64(), doctor, testData, t)

	// Started from a favorite treatment plan
	tpNew := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentID:   tp.ID,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   ftp.ID,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != ftp.Note {
		t.Fatalf("Expected '%s' got '%s'", ftp.Note, tpNew.TreatmentPlan.Note)
	}

	// TP shouldn't have deviated
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if tp.ContentSource.Deviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to same content (should not mark TP as deviated)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.ID.Int64(), ftp.Note))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if tp.ContentSource.Deviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to different content (should deviate)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.ID.Int64(), "something else"))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if !tp.ContentSource.Deviated {
		t.Fatal("treatment plan should have deviated")
	}
}
