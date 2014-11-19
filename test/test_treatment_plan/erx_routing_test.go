package test_treatment_plan

import (
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestERXRouting_RXStarted(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// add a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil",
		TreatmentPlanId:  tp.Id,
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
	}

	treatmentsResponse := test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), tp.Id.Int64(), t)

	// submit the treatment plan back to the patient
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// now lets act as though we are in a state where the patient and the treatments have all the erx information needed
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	erxPatientId := int64(100)
	treatmentPrescriptionId := int64(105)
	err = testData.DataApi.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), erxPatientId)
	test.OK(t, err)

	treatmentsResponse.TreatmentList.Treatments[0].ERx = &common.ERxData{
		PrescriptionId: encoding.NewObjectId(treatmentPrescriptionId),
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceId:     12345,
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection)
	test.OK(t, err)

	err = testData.DataApi.StartRXRoutingForTreatmentsAndTreatmentPlan(treatmentsResponse.TreatmentList.Treatments, pharmacySelection, tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)

	// at this point the treatment plan is in the rx started state
	// lets go ahead and call the worker to complete the rest of the steps to ensure that its successfully able to activate the treatment plan
	// after routing the prescriptions
	stubERxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubERxAPI.PrescriptionIdToPrescriptionStatuses = map[int64][]common.StatusEvent{
		treatmentPrescriptionId: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_ENTERED,
		},
		},
	}
	doctor_treatment_plan.StartWorker(testData.DataApi, stubERxAPI, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// at this point the treatment plan should be activated
	treatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, common.TPStatusActive, treatmentPlan.Status)

	// there should also be a case message for the patient
	caseMessages, err := testData.DataApi.ListCaseMessages(treatmentPlan.PatientCaseId.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
}

func TestERXRouting_RXSent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// add a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil",
		TreatmentPlanId:  tp.Id,
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
	}

	treatmentsResponse := test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), tp.Id.Int64(), t)

	// submit the treatment plan back to the patient
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// now lets act as though we are in a state where the patient and the treatments have all the erx information needed
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	erxPatientId := int64(100)
	treatmentPrescriptionId := int64(105)
	err = testData.DataApi.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), erxPatientId)
	test.OK(t, err)

	treatmentsResponse.TreatmentList.Treatments[0].ERx = &common.ERxData{
		PrescriptionId: encoding.NewObjectId(treatmentPrescriptionId),
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceId:     12345,
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection)
	test.OK(t, err)

	err = testData.DataApi.StartRXRoutingForTreatmentsAndTreatmentPlan(treatmentsResponse.TreatmentList.Treatments, pharmacySelection, tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)

	// at this point the treatment plan is in the rx started state
	// lets go ahead and call the worker to complete the rest of the steps to ensure that its successfully able to activate the treatment plan
	// after routing the prescriptions
	stubERxAPI := testData.Config.ERxAPI.(*erx.StubErxService)
	stubERxAPI.PrescriptionIdToPrescriptionStatuses = map[int64][]common.StatusEvent{
		treatmentPrescriptionId: []common.StatusEvent{common.StatusEvent{
			Status: api.ERX_STATUS_SENT,
		},
		},
	}
	doctor_treatment_plan.StartWorker(testData.DataApi, stubERxAPI, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// at this point the treatment plan should be activated
	treatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, common.TPStatusActive, treatmentPlan.Status)

	// there should also be a case message for the patient
	caseMessages, err := testData.DataApi.ListCaseMessages(treatmentPlan.PatientCaseId.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
}

func TestERxRouting_CaseMessageExistsAlready(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.Config.ERxRouting = true
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	_, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// add a treatment
	treatment1 := &common.Treatment{
		DrugInternalName: "Advil",
		TreatmentPlanId:  tp.Id,
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
	}

	treatmentsResponse := test_integration.AddAndGetTreatmentsForPatientVisit(testData, []*common.Treatment{treatment1}, doctor.AccountId.Int64(), tp.Id.Int64(), t)

	// submit the treatment plan back to the patient
	test_integration.SubmitPatientVisitBackToPatient(tp.Id.Int64(), doctor, testData, t)

	// now lets act as though we are in a state where the patient and the treatments have all the erx information needed
	patient, err := testData.DataApi.GetPatientFromId(tp.PatientId)
	test.OK(t, err)

	erxPatientId := int64(100)
	err = testData.DataApi.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), erxPatientId)
	test.OK(t, err)

	treatmentsResponse.TreatmentList.Treatments[0].ERx = &common.ERxData{
		PrescriptionId: encoding.NewObjectId(105),
	}

	pharmacySelection := &pharmacy.PharmacyData{
		SourceId:     12345,
		Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		AddressLine1: "12345 Marin Street",
		City:         "San Francisco",
		State:        "CA",
		Phone:        "12345667",
	}

	err = testData.DataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), pharmacySelection)
	test.OK(t, err)

	err = testData.DataApi.StartRXRoutingForTreatmentsAndTreatmentPlan(treatmentsResponse.TreatmentList.Treatments, pharmacySelection, tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)

	// now lets go ahead and activate the treatment plan as well as send the case message for the patient
	err = testData.DataApi.ActivateTreatmentPlan(tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)

	caseMessage := &common.CaseMessage{
		CaseID:   tp.PatientCaseId.Int64(),
		PersonID: doctor.PersonId,
		Body:     "foo",
		Attachments: []*common.CaseMessageAttachment{
			&common.CaseMessageAttachment{
				ItemType: common.AttachmentTypeTreatmentPlan,
				ItemID:   tp.Id.Int64(),
			},
		},
	}
	_, err = testData.DataApi.CreateCaseMessage(caseMessage)
	test.OK(t, err)

	// now lets go ahead and get the worker to consume the message
	doctor_treatment_plan.StartWorker(testData.DataApi, testData.Config.ERxAPI, testData.Config.Dispatcher, testData.Config.ERxRoutingQueue, testData.Config.ERxStatusQueue, 0, metrics.NewRegistry())

	// at this point the treatment plan should be activated
	treatmentPlan, err := testData.DataApi.GetAbridgedTreatmentPlan(tp.Id.Int64(), doctor.DoctorId.Int64())
	test.OK(t, err)
	test.Equals(t, common.TPStatusActive, treatmentPlan.Status)

	// there should also be just a single case message for the patient
	caseMessages, err := testData.DataApi.ListCaseMessages(treatmentPlan.PatientCaseId.Int64(), api.PATIENT_ROLE)
	test.OK(t, err)
	test.Equals(t, 1, len(caseMessages))
}
