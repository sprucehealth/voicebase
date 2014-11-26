package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/apiclient"
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
	_, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	note := "Dear foo, this is my message"
	if err := cli.UpdateTreatmentPlanNote(treatmentPlan.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}
	if tp, err := cli.TreatmentPlan(treatmentPlan.Id.Int64(), false); err != nil {
		t.Fatal(err)
	} else if tp.Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, tp.Note)
	}

	// Update treatment plan message
	note = "Dear foo, I have changed my mind"
	if err := cli.UpdateTreatmentPlanNote(treatmentPlan.Id.Int64(), note); err != nil {
		t.Fatal(err)
	}

	if tp, err := cli.TreatmentPlan(treatmentPlan.Id.Int64(), false); err != nil {
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

	if ftp := CreateFTPFromTP(tp, "test FTP 1", testData, doctor, t); ftp.Note != "" {
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
	if ftp := CreateFTPFromTP(tp, "test FTP 2", testData, doctor, t); ftp.Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, ftp.Note)
	}

	// (test get FTP response)
	if ftps, err := cli.ListFavoriteTreatmentPlans(); err != nil {
		t.Fatal(err)
	} else if len(ftps) != 1 {
		t.Fatalf("Expected 1 ftp got %d", len(ftps))
	} else if ftps[0].Note != note {
		t.Fatalf("Expected '%s' got '%s'", note, ftps[0].Note)
	}
}
