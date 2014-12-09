package test_integration

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/test"
)

func TestMedicalRecordWorker(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	doctorID := GetDoctorIDOfCurrentDoctor(testData, t)
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)

	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	treatments := []*common.Treatment{
		&common.Treatment{
			DrugInternalName: "Advil",
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
		},
	}
	testData.DataAPI.AddTreatmentsForTreatmentPlan(treatments, doctorID, treatmentPlan.ID.Int64(), patient.PatientID.Int64())

	SubmitPatientVisitBackToPatient(treatmentPlan.ID.Int64(), doctor, testData, t)

	signer := &common.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	worker := medrecord.StartWorker(testData.DataAPI, testData.Config.MedicalRecordQueue, testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, store, 60)
	defer worker.Stop()

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientRequestMedicalRecordURLPath,
		"application/json", bytes.NewReader([]byte("{}")), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	emailService := testData.Config.EmailService.(*email.TestService)

	var email []*email.TestTemplated
	for i := 0; i < 10; i++ {
		_, email = emailService.Reset()
		if len(email) != 0 {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}
	t.Logf("%+v", email[0])
}

func TestMedicalRecordWorker_VisitOpen(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create a visit in the open state with no questions answered
	pr := SignupRandomTestPatient(t, testData)
	pv := CreatePatientVisitForPatient(pr.Patient.PatientID.Int64(), testData, t)

	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	signer := &common.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	worker := medrecord.StartWorker(testData.DataAPI, testData.Config.MedicalRecordQueue, testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, store, 60)
	defer worker.Stop()

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientRequestMedicalRecordURLPath,
		"application/json", bytes.NewReader([]byte("{}")), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	emailService := testData.Config.EmailService.(*email.TestService)

	var email []*email.TestTemplated
	for i := 0; i < 5; i++ {
		_, email = emailService.Reset()
		if len(email) != 0 {
			break
		}
		time.Sleep(time.Millisecond * 200)
	}
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}
	t.Logf("%+v", email[0])
}
