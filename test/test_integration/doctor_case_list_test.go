package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestDoctorCaseList(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Setup

	mr, _, _ := SignupRandomTestMA(t, testData)
	ma, err := testData.DataApi.GetDoctorFromId(mr.DoctorId)
	test.OK(t, err)
	doctorID := GetDoctorIdOfCurrentDoctor(testData, t)
	doctor, err := testData.DataApi.GetDoctorFromId(doctorID)
	test.OK(t, err)
	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)
	caseID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)

	doctorCli := DoctorClient(testData, t, doctorID)
	maCli := DoctorClient(testData, t, ma.DoctorId.Int64())
	patientCli := PatientClient(testData, t, patient.PatientId.Int64())

	test.OK(t, doctorCli.UpdateTreatmentPlanNote(treatmentPlan.Id.Int64(), "foo"))
	test.OK(t, doctorCli.SubmitTreatmentPlan(treatmentPlan.Id.Int64()))

	// Treatment plan submitted (and denorm field checks)

	// Doctor
	fitems, err := doctorCli.DoctorCaseList()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	t.Logf("%+v", fitems[0])
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)
	test.Equals(t, false, fitems[0].EventTime == 0)
	test.Equals(t, "Treatment plan completed by Dr. Test LastName", fitems[0].EventDescription)
	// MA
	fitems, err = maCli.DoctorCaseList()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	t.Logf("%+v", fitems[0])
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)
	test.Equals(t, false, fitems[0].EventTime == 0)
	test.Equals(t, "Treatment plan completed by Dr. Test LastName", fitems[0].EventDescription)

	// Message from doctor

	_, err = doctorCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)
	items, err := testData.DataApi.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	t.Logf("%+v", items[0])
	test.Equals(t, "Message by Dr. Test LastName", items[0].LastEvent)

	// Message from patient

	_, err = patientCli.PostCaseMessage(caseID, "bar", nil)
	test.OK(t, err)
	items, err = testData.DataApi.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	t.Logf("%+v", items[0])
	test.Equals(t, "Message by Test Test", items[0].LastEvent)

	// MA assigns case

	_, err = maCli.AssignCase(caseID, "assign", nil)
	test.OK(t, err)
	items, err = testData.DataApi.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	t.Logf("%+v", items[0])
	test.Equals(t, "Assigned to Dr. Test LastName", items[0].LastEvent)

	// Test multiple doctors and cases

	dr, _, _ := SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)
	visit, treatmentPlan = CreateRandomPatientVisitAndPickTP(t, testData, doctor2)
	case2ID, err := testData.DataApi.GetPatientCaseIdFromPatientVisitId(visit.PatientVisitId)
	test.OK(t, err)
	doctor2Cli := DoctorClient(testData, t, dr.DoctorId)
	test.OK(t, doctor2Cli.UpdateTreatmentPlanNote(treatmentPlan.Id.Int64(), "foo"))
	test.OK(t, doctor2Cli.SubmitTreatmentPlan(treatmentPlan.Id.Int64()))

	// Each doctor should only see their cases

	items, err = testData.DataApi.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, caseID, items[0].CaseID)
	items, err = testData.DataApi.PatientCaseFeedForDoctor(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, case2ID, items[0].CaseID)

	// MA should see all cases

	items, err = testData.DataApi.PatientCaseFeed()
	test.OK(t, err)
	test.Equals(t, 2, len(items))
}
