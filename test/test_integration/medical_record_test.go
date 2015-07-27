package test_integration

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/test"
)

type diagnosisTestSvc struct {
	diagnosis.API
	codes map[string]*diagnosis.Diagnosis
}

func (d *diagnosisTestSvc) DiagnosisForCodeIDs(codeIDs []string) (map[string]*diagnosis.Diagnosis, error) {
	res := make(map[string]*diagnosis.Diagnosis, len(codeIDs))
	for _, cid := range codeIDs {
		res[cid] = d.codes[cid]
	}
	return res, nil
}

func TestMedicalRecordWorker(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	AddTreatmentsToTreatmentPlan(treatmentPlan.ID.Int64(), doctor, t, testData)
	_, guideIDs := CreateTestResourceGuides(t, testData)
	test.OK(t, testData.DataAPI.AddResourceGuidesToTreatmentPlan(treatmentPlan.ID.Int64(), guideIDs))
	SubmitPatientVisitDiagnosis(visit.PatientVisitID, doctor, testData, t)
	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	diagSvc := &diagnosisTestSvc{
		codes: map[string]*diagnosis.Diagnosis{},
	}

	signer := &sig.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	mediaStore := media.NewStore("http://example.com", signer, store)
	worker := medrecord.NewWorker(
		testData.DataAPI, diagSvc, testData.Config.MedicalRecordQueue,
		testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, mediaStore, 60, nil)

	mrID, err := PatientClient(testData, t, patient.ID.Int64()).RequestMedicalRecord()
	test.OK(t, err)

	emailService := testData.Config.EmailService.(*email.TestService)

	worker.Do()

	email := emailService.Reset()
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}

	_, _, err = store.Get(fmt.Sprintf("%d.html", mrID))
	test.OK(t, err)
}

func TestMedicalRecordWorker_VisitOpen(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create a visit in the open state with no questions answered
	pr := SignupRandomTestPatient(t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.ID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	diagSvc := &diagnosisTestSvc{
		codes: map[string]*diagnosis.Diagnosis{},
	}

	signer := &sig.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	mediaStore := media.NewStore("http://example.com", signer, store)
	worker := medrecord.NewWorker(
		testData.DataAPI, diagSvc, testData.Config.MedicalRecordQueue,
		testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, mediaStore, 60, nil)

	_, err = PatientClient(testData, t, patient.ID.Int64()).RequestMedicalRecord()
	test.OK(t, err)

	emailService := testData.Config.EmailService.(*email.TestService)

	worker.Do()

	email := emailService.Reset()
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}
}
