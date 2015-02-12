package test_integration

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
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
			DrugInternalName: "Drug1 (Route1 - Form1)",
			DosageStrength:   "Strength1",
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

	signer := &sig.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	mediaStore := media.NewStore("http://example.com", signer, store)
	worker := medrecord.NewWorker(testData.DataAPI, testData.Config.MedicalRecordQueue, testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, mediaStore, 60)

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientRequestMedicalRecordURLPath,
		"application/json", bytes.NewReader([]byte("{}")), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	emailService := testData.Config.EmailService.(*email.TestService)

	worker.Do()

	var email []*email.TestTemplated
	_, email = emailService.Reset()
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}
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

	signer := &sig.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	mediaStore := media.NewStore("http://example.com", signer, store)
	worker := medrecord.NewWorker(testData.DataAPI, testData.Config.MedicalRecordQueue, testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, mediaStore, 60)

	res, err := testData.AuthPost(testData.APIServer.URL+apipaths.PatientRequestMedicalRecordURLPath,
		"application/json", bytes.NewReader([]byte("{}")), patient.AccountID.Int64())
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)

	emailService := testData.Config.EmailService.(*email.TestService)

	worker.Do()

	var email []*email.TestTemplated
	_, email = emailService.Reset()
	if len(email) == 0 {
		t.Fatal("Did not receive medical record email")
	}
}
