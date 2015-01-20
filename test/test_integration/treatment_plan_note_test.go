package test_integration

import (
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

func TestTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
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

func TestFavoriteTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dres.DoctorID)
	test.OK(t, err)

	cli := DoctorClient(testData, t, dres.DoctorID)

	// Create a patient treatment plan, and save a draft message
	pv, tp0 := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	AddTreatmentsToTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)
	AddRegimenPlanForTreatmentPlan(tp0.ID.Int64(), doctor, t, testData)
	// Refetch the treatment plan to fill in regimen steps and treatments
	tp, err := cli.TreatmentPlan(tp0.ID.Int64(), false, doctor_treatment_plan.AllSections)
	test.OK(t, err)

	// A FTP created from a TP with an empty note should also have an empty note

	ftpTemplate := &doctor_treatment_plan.FavoriteTreatmentPlan{
		Name:          "Test FTP",
		RegimenPlan:   tp.RegimenPlan,
		TreatmentList: tp.TreatmentList,
		Note:          tp.Note,
	}
	if ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftpTemplate, tp.ID.Int64()); err != nil {
		t.Fatal(err)
	} else if ftp.Note != "" {
		t.Fatalf("Expected an empty note got '%s'", ftp.Note)
	} else {
		// Delete the FTP to avoid conflicting with tests below
		test.OK(t, cli.DeleteFavoriteTreatmentPlan(ftp.ID.Int64()))
	}

	// A FTP created from a TP with a non-empty note should have the same note

	// (test create FTP response)
	note := "Dear foo, I have changed my mind"
	if err := cli.UpdateTreatmentPlanNote(tp.ID.Int64(), note); err != nil {
		t.Fatal(err)
	}
	tp.Note = note
	ftpTemplate.Note = tp.Note
	if ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftpTemplate, tp.ID.Int64()); err != nil {
		t.Fatal(err)
	} else if ftp.Note != tp.Note {
		t.Fatalf("Expected '%s' got '%s'", tp.Note, ftp.Note)
	}

	// (test get FTP response)
	if ftps, err := cli.ListFavoriteTreatmentPlans(); err != nil {
		t.Fatal(err)
	} else if len(ftps) != 1 {
		t.Fatalf("Expected 1 ftp got %d", len(ftps))
	} else if ftps[0].Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, ftps[0].Note)
	}

	// Old deprecated endpoint
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DeprecatedDoctorSavedMessagesURLPath, doctor.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	test.OK(t, err)
	test.Equals(t, "{\"message\":\"Dear foo, I have changed my mind\"}\n", string(b))

	tp2 := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentType: common.TPParentTypePatientVisit,
		ParentID:   encoding.NewObjectID(pv.PatientVisitID),
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeTreatmentPlan,
		ID:   tp.ID,
	}, doctor, testData, t)

	res, err = testData.AuthGet(
		testData.APIServer.URL+apipaths.DeprecatedDoctorSavedMessagesURLPath+"?treatment_plan_id="+strconv.FormatInt(tp2.TreatmentPlan.ID.Int64(), 10),
		doctor.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	b, err = ioutil.ReadAll(res.Body)
	test.OK(t, err)
	test.Equals(t, true, strings.Contains(string(b), "Here is your revised treatment plan."))

}

func TestVersionedTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
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
	defer testData.Close()
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
	} else if tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to same content (should not mark TP as deviated)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.ID.Int64(), ftp.Note))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to different content (should deviate)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.ID.Int64(), "something else"))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.ID.Int64(), true, doctor_treatment_plan.NoSections); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if !tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should have deviated")
	}
}
