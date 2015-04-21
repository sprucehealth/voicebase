package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestDoctorCaseHistory(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Setup

	mr, _, _ := SignupRandomTestMA(t, testData)
	ma, err := testData.DataAPI.GetDoctorFromID(mr.DoctorID)
	test.OK(t, err)

	dr, _, _ := SignupRandomTestDoctor(t, testData)
	doctorID := dr.DoctorID
	doctor, err := testData.DataAPI.GetDoctorFromID(doctorID)
	test.OK(t, err)
	visit, treatmentPlan := CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)
	caseID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)

	doctorCli := DoctorClient(testData, t, doctorID)
	maCli := DoctorClient(testData, t, ma.DoctorID.Int64())
	patientCli := PatientClient(testData, t, patient.PatientID.Int64())

	test.OK(t, doctorCli.UpdateTreatmentPlanNote(treatmentPlan.ID.Int64(), "foo"))
	test.OK(t, doctorCli.SubmitTreatmentPlan(treatmentPlan.ID.Int64()))

	// Treatment plan submitted (and denorm field checks)

	// Doctor
	fitems, err := doctorCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)
	test.Equals(t, false, fitems[0].EventTime == 0)
	test.Equals(t, "Treatment plan completed by Dr. Test LastName", fitems[0].EventDescription)
	// MA
	fitems, err = maCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)
	test.Equals(t, false, fitems[0].EventTime == 0)
	test.Equals(t, "Treatment plan completed by Dr. Test LastName", fitems[0].EventDescription)

	// Message from doctor

	_, err = doctorCli.PostCaseMessage(caseID, "foo", nil)
	test.OK(t, err)
	items, err := testData.DataAPI.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, "Message by Dr. Test LastName", items[0].LastEvent)

	// Message from patient

	_, err = patientCli.PostCaseMessage(caseID, "bar", nil)
	test.OK(t, err)
	items, err = testData.DataAPI.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, "Message by Dr. Test LastName", items[0].LastEvent)

	// MA assigns case

	_, err = maCli.AssignCase(caseID, "assign", nil)
	test.OK(t, err)
	items, err = testData.DataAPI.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, "Assigned to Dr. Test LastName", items[0].LastEvent)

	// Test multiple doctors and cases

	dr, _, _ = SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	visit, treatmentPlan = CreateRandomPatientVisitAndPickTP(t, testData, doctor2)
	case2ID, err := testData.DataAPI.GetPatientCaseIDFromPatientVisitID(visit.PatientVisitID)
	test.OK(t, err)
	doctor2Cli := DoctorClient(testData, t, dr.DoctorID)
	test.OK(t, doctor2Cli.UpdateTreatmentPlanNote(treatmentPlan.ID.Int64(), "foo"))
	test.OK(t, doctor2Cli.SubmitTreatmentPlan(treatmentPlan.ID.Int64()))

	// Each doctor should only see their cases

	items, err = testData.DataAPI.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, caseID, items[0].CaseID)
	items, err = testData.DataAPI.PatientCaseFeedForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, case2ID, items[0].CaseID)

	// MA should see all cases

	items, err = testData.DataAPI.PatientCaseFeed()
	test.OK(t, err)
	test.Equals(t, 2, len(items))
}
