package test_integration

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
)

func TestDoctorCaseHistory(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// Setup

	mr, _, _ := SignupRandomTestCC(t, testData, true)
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
	maCli := DoctorClient(testData, t, ma.ID.Int64())
	patientCli := PatientClient(testData, t, patient.ID.Int64())

	test.OK(t, doctorCli.UpdateTreatmentPlanNote(treatmentPlan.ID.Int64(), "foo"))
	test.OK(t, doctorCli.SubmitTreatmentPlan(treatmentPlan.ID.Int64()))

	// Treatment plan submitted (and denorm field checks)

	// Doctor
	fitems, err := doctorCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test LastName", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)
	// MA
	fitems, err = maCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	test.Equals(t, "Test", fitems[0].PatientFirstName)
	test.Equals(t, "Dr. Test LastName", fitems[0].LastVisitDoctor)
	test.Equals(t, false, fitems[0].LastVisitTime == 0)

	// Unsubmitted visit shouldn't show up in the feed
	newVisit, err := patientCli.CreatePatientVisit(api.AcnePathwayTag, doctorID, SetupTestHeaders())
	test.OK(t, err)
	_ = newVisit
	fitems, err = doctorCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))
	fitems, err = maCli.DoctorCaseHistory()
	test.OK(t, err)
	test.Equals(t, 1, len(fitems))

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

	items, err := testData.DataAPI.PatientCaseFeedForDoctor(doctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, caseID, items[0].CaseID)
	items, err = testData.DataAPI.PatientCaseFeedForDoctor(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	test.Equals(t, case2ID, items[0].CaseID)

	// MA should see all cases  or a filtered set depending on params
	s := time.Unix(1, 0)
	now := time.Now().Add(time.Minute)
	items, err = testData.DataAPI.PatientCaseFeed(nil, &s, &now)
	test.OK(t, err)
	test.Equals(t, 2, len(items))
	items, err = testData.DataAPI.PatientCaseFeed([]int64{case2ID}, &s, nil)
	test.OK(t, err)
	test.Equals(t, 1, len(items))
	items, err = testData.DataAPI.PatientCaseFeed([]int64{case2ID}, &s, &s)
	test.OK(t, err)
	test.Equals(t, 0, len(items))
	items, err = testData.DataAPI.PatientCaseFeed(nil, nil, nil)
	test.OK(t, err)
	test.Equals(t, 2, len(items))
}
