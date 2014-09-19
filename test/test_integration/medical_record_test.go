package test_integration

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
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

	doctorID := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)

	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)
	SubmitPatientVisitBackToPatient(treatmentPlan.Id.Int64(), doctor, testData, t)

	treatments := []*common.Treatment{
		&common.Treatment{
			DrugInternalName: "Advil",
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
		},
	}
	testData.DataApi.AddTreatmentsForTreatmentPlan(treatments, doctorID, treatmentPlan.Id.Int64(), patient.PatientId.Int64())

	signer := &common.Signer{}
	store := testData.Config.Stores.MustGet("medicalrecords")
	worker := medrecord.StartWorker(testData.DataApi, testData.Config.MedicalRecordQueue, testData.Config.EmailService, "from@somewhere.com",
		"apidomain", "webdomain", signer, store, store, 60)
	defer worker.Stop()

	res, err := testData.AuthPost(testData.APIServer.URL+router.PatientRequestMedicalRecordURLPath,
		"application/json", bytes.NewReader([]byte("{}")), patient.AccountId.Int64())
	test.OK(t, err)
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
