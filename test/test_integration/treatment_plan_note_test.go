package test_integration

import (
	"io/ioutil"
	"testing"

	"github.com/sprucehealth/backend/apiclient"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/test"
)

func TestTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	cli := &apiclient.DoctorClient{
		BaseURL:   testData.APIServer.URL,
		AuthToken: dres.Token,
	}

	// Create a patient treatment plan, and save a draft message
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}
	if tp, err := cli.TreatmentPlan(tp.Id.Int64(), false); err != nil {
		t.Fatal(err)
	} else if tp.Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, tp.Note)
	}

	// Update treatment plan message
	note = "Dear foo, I have changed my mind"
	if err := cli.UpdateTreatmentPlanNote(tp.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}

	if tp, err := cli.TreatmentPlan(tp.Id.Int64(), false); err != nil {
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
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	cli := &apiclient.DoctorClient{
		BaseURL:   testData.APIServer.URL,
		AuthToken: dres.Token,
	}

	// Create a patient treatment plan, and save a draft message
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	AddTreatmentsToTreatmentPlan(tp.Id.Int64(), doctor, t, testData)
	AddRegimenPlanForTreatmentPlan(tp.Id.Int64(), doctor, t, testData)
	// Refetch the treatment plan to fill in regimen steps and treatments
	tp, err = cli.TreatmentPlan(tp.Id.Int64(), false)
	test.OK(t, err)

	// A FTP created from a TP with an empty note should also have an empty note

	ftpTemplate := &common.FavoriteTreatmentPlan{
		Name:          "Test FTP",
		RegimenPlan:   tp.RegimenPlan,
		TreatmentList: tp.TreatmentList,
		Note:          tp.Note,
	}
	if ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftpTemplate, tp.Id.Int64()); err != nil {
		t.Fatal(err)
	} else if ftp.Note != "" {
		t.Fatalf("Expected an empty note got '%s'", ftp.Note)
	} else {
		// Delete the FTP to avoid conflicting with tests below
		test.OK(t, cli.DeleteFavoriteTreatmentPlan(ftp.Id.Int64()))
	}

	// A FTP created from a TP with a non-empty note should have the same note

	// (test create FTP response)
	note := "Dear foo, I have changed my mind"
	if err := cli.UpdateTreatmentPlanNote(tp.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}
	tp.Note = note
	ftpTemplate.Note = tp.Note
	if ftp, err := cli.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftpTemplate, tp.Id.Int64()); err != nil {
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
	res, err := testData.AuthGet(testData.APIServer.URL+apipaths.DeprecatedDoctorSavedMessagesURLPath, doctor.AccountId.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	test.OK(t, err)
	test.Equals(t, "{\"message\":\"Dear foo, I have changed my mind\"}\n", string(b))
}

func TestVersionedTreatmentPlanNote(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dres, _, _ := SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	cli := &apiclient.DoctorClient{
		BaseURL:   testData.APIServer.URL,
		AuthToken: dres.Token,
	}

	// Create a patient treatment plan and set the note
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}
	ftp := CreateFavoriteTreatmentPlan(tp.Id.Int64(), testData, doctor, t)

	SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// Started from scratch
	tpNew := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   tp.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, nil, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != "" {
		t.Fatalf("Expected empty note got '%s'", tpNew.TreatmentPlan.Note)
	}

	// Started from a treatment plan
	tpNew = PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   tp.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeTreatmentPlan,
		ID:   tp.Id,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != doctor_treatment_plan.VersionedTreatmentPlanNote {
		t.Fatalf("Expected '%s' got '%s'", doctor_treatment_plan.VersionedTreatmentPlanNote, tpNew.TreatmentPlan.Note)
	}

	// Started from a favorite treatment plan
	tpNew = PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   tp.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   ftp.Id,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != ftp.Note {
		t.Fatalf("Expected '%s' got '%s'", ftp.Note, tpNew.TreatmentPlan.Note)
	}

	// Make sure note is maintained after submitting
	SubmitPatientVisitBackToPatient(tpNew.TreatmentPlan.Id.Int64(), doctor, testData, t)
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.Id.Int64(), false); err != nil {
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
	doctor, err := testData.DataApi.GetDoctorFromId(dres.DoctorId)
	test.OK(t, err)

	cli := &apiclient.DoctorClient{
		BaseURL:   testData.APIServer.URL,
		AuthToken: dres.Token,
	}

	// Create a patient treatment plan and set the note
	_, tp := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(tp.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}
	ftp := CreateFavoriteTreatmentPlan(tp.Id.Int64(), testData, doctor, t)
	SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// Started from a favorite treatment plan
	tpNew := PickATreatmentPlan(&common.TreatmentPlanParent{
		ParentId:   tp.Id,
		ParentType: common.TPParentTypeTreatmentPlan,
	}, &common.TreatmentPlanContentSource{
		Type: common.TPContentSourceTypeFTP,
		ID:   ftp.Id,
	}, doctor, testData, t)
	if tpNew.TreatmentPlan.Note != ftp.Note {
		t.Fatalf("Expected '%s' got '%s'", ftp.Note, tpNew.TreatmentPlan.Note)
	}

	// TP shouldn't have deviated
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.Id.Int64(), true); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to same content (should not mark TP as deviated)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.Id.Int64(), ftp.Note))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.Id.Int64(), true); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should not have deviated")
	}

	// Update note to different content (should deviate)
	test.OK(t, cli.UpdateTreatmentPlanNote(tpNew.TreatmentPlan.Id.Int64(), "something else"))
	if tp, err := cli.TreatmentPlan(tpNew.TreatmentPlan.Id.Int64(), true); err != nil {
		t.Fatal(err)
	} else if tp.ContentSource == nil {
		t.Fatal("ContentShould should not be nil")
	} else if !tp.ContentSource.HasDeviated {
		t.Fatal("treatment plan should have deviated")
	}
}
