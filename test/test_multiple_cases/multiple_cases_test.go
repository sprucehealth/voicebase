package test_multiple_cases

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that in the situation where a patient
// submits two indepnedent cases to the first available doctor,
// doctorA cannot open caseB but can see caseB in the list.
func TestMultipleCases_JBCQ_JBCQ(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// have a single patient submit two different cases with the first available
	// doctor picked for both cases
	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor1, err := testData.DataAPI.GetDoctorFromID(dr1.DoctorID)
	test.OK(t, err)
	pv1 := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pv1.PatientVisitID)
	test.OK(t, err)

	pathway := test_integration.CreatePathway(t, testData, "test")
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor2, err := testData.DataAPI.GetDoctorFromID(dr2.DoctorID)
	test.OK(t, err)
	pv2, _ := test_integration.CreateRandomPatientVisitAndPickTPForPathway(t, testData, pathway, patient, doctor2)

	// attempt to open the patient file for doctor1
	dc1 := test_integration.DoctorClient(testData, t, doctor1.DoctorID.Int64())

	// lets get doctor1 to claim case1
	pc, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv1.PatientVisitID)
	test.OK(t, err)
	test.OK(t, dc1.ClaimCase(pc.ID.Int64()))

	cases, err := dc1.CasesForPatient(patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(cases))

	// doctor1 should be able to open visit for case1 and for case2
	_, err = dc1.ReviewVisit(pv1.PatientVisitID)
	test.OK(t, err)

	_, err = dc1.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)

	// doctor2 should be able to open visit for case2 but not for case1
	dc2 := test_integration.DoctorClient(testData, t, doctor2.DoctorID.Int64())
	cases, err = dc2.CasesForPatient(patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	_, err = dc2.ReviewVisit(pv1.PatientVisitID)
	test.Equals(t, true, err != nil)
	_, err = dc2.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)

	// now lets get doctor1 to submit a TP and then see if doctor2 can query the visit for case1
	tp1, err := dc1.PickTreatmentPlanForVisit(pv1.PatientVisitID, nil)
	test.OK(t, err)
	test.OK(t, dc1.UpdateTreatmentPlanNote(tp1.ID.Int64(), "foo"))
	test.OK(t, dc1.SubmitTreatmentPlan(tp1.ID.Int64()))
	_, err = dc2.ReviewVisit(pv1.PatientVisitID)
	test.OK(t, err)
}

// This test is to ensure that an unsubmitted case cannot be seen
// by the doctor in the patient file
func TestMultipleCases_Submitted_Started(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// have a single patient submit 1 case, and then start but not complete another case
	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.PatientID.Int64())

	// unsubmitted case
	_, err := pc.CreatePatientVisit(api.AcnePathwayTag, 0, test_integration.SetupTestHeaders())
	test.OK(t, err)

	// submitted case
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pathway := test_integration.CreatePathway(t, testData, "test")
	_, tp := test_integration.CreateRandomPatientVisitAndPickTPForPathway(t, testData, pathway, pr.Patient, doctor)

	dc := test_integration.DoctorClient(testData, t, dr.DoctorID)
	cases, err := dc.CasesForPatient(pr.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(cases))
	test.Equals(t, tp.PatientCaseID.Int64(), cases[0].ID)
}

// This test is to ensure that if the patient submits two cases to the unassigned queue
// the same doctor is able to pick up and complete both
func TestMultipleCases_SameDoctorPicksUpCase(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// submit acne case
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pv1, tp1 := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	patient, err := testData.DataAPI.GetPatientFromID(tp1.PatientID)
	test.OK(t, err)

	// submit test pathway case
	pathway := test_integration.CreatePathway(t, testData, "test")
	pv2, tp2 := test_integration.CreateRandomPatientVisitAndPickTPForPathway(t, testData, pathway, patient, doctor)

	dc := test_integration.DoctorClient(testData, t, dr.DoctorID)

	// ensure that doctor can open both
	_, err = dc.ReviewVisit(pv1.PatientVisitID)
	test.OK(t, err)
	_, err = dc.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)

	// ensure that doctor can submit TP for both
	test.OK(t, dc.UpdateTreatmentPlanNote(tp1.ID.Int64(), "foo"))
	test.OK(t, dc.SubmitTreatmentPlan(tp1.ID.Int64()))
	test.OK(t, dc.UpdateTreatmentPlanNote(tp2.ID.Int64(), "foo"))
	test.OK(t, dc.SubmitTreatmentPlan(tp2.ID.Int64()))

	// ensure that there are 2 completed items in doctor's queue
	completedItems, err := dc.History()
	test.OK(t, err)
	test.Equals(t, 2, len(completedItems))
}

// This test is to ensure that if a patient submits a case with doctors picked
// then the two doctors cannot mistakely modify each others cases.
func TestMultipleCases_DoctorsAssigned(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// submit acne case assigned to doctor
	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.PatientID.Int64())
	pv, err := pc.CreatePatientVisit(api.AcnePathwayTag, dr.DoctorID, test_integration.SetupTestHeaders())
	test.OK(t, err)

	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	pathway := test_integration.CreatePathway(t, testData, "test")
	test_integration.UploadLayoutPairForPathway(pathway.Tag, testData, t)
	careProvoidingStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", pathway.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvoidingStateID, dr2.DoctorID))
	pv2, err := pc.CreatePatientVisit(pathway.Tag, dr2.DoctorID, test_integration.SetupTestHeaders())
	test.OK(t, err)

	// Both cases should be claimed
	case1, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, true, case1.Claimed)

	case2, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv2.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, true, case2.Claimed)

	// Update patient case state to be active so that we can proceed forward as though the cases
	// were submitted
	nextStatus := common.PCStatusActive
	test.OK(t, testData.DataAPI.UpdatePatientCase(case1.ID.Int64(), &api.PatientCaseUpdate{
		Status: &nextStatus,
	}))

	test.OK(t, testData.DataAPI.UpdatePatientCase(case2.ID.Int64(), &api.PatientCaseUpdate{
		Status: &nextStatus,
	}))

	// both doctors should be able to open each others cases
	dc1 := test_integration.DoctorClient(testData, t, dr.DoctorID)
	_, err = dc1.ReviewVisit(pv.PatientVisitID)
	test.OK(t, err)
	_, err = dc1.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)

	dc2 := test_integration.DoctorClient(testData, t, dr2.DoctorID)
	_, err = dc2.ReviewVisit(pv.PatientVisitID)
	test.OK(t, err)
	_, err = dc2.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)

	// doctor1 should not be able to modify doctor2's case
	_, err = dc1.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	_, err = dc1.PickTreatmentPlanForVisit(pv2.PatientVisitID, nil)
	test.Equals(t, true, err != nil)

	// doctor2 should not be able to modify doctor1's case
	_, err = dc2.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.Equals(t, true, err != nil)
	_, err = dc2.PickTreatmentPlanForVisit(pv2.PatientVisitID, nil)
	test.OK(t, err)
}

// This test is to ensure that if a patient submits a case into the unassigned queue and another case
// assigned to a doctor, then the interaction works as expected
func TestMultipleCases_JBCQ_Assigned(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// submit a case to the unassigned queue
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)
	pv, tp := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)

	// submit a case by preselecting a doctor
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	pathway := test_integration.CreatePathway(t, testData, "test")
	test_integration.UploadLayoutPairForPathway(pathway.Tag, testData, t)
	careProvoidingStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", pathway.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvoidingStateID, dr2.DoctorID))
	pc := test_integration.PatientClient(testData, t, tp.PatientID)
	pv2, err := pc.CreatePatientVisit(pathway.Tag, dr2.DoctorID, test_integration.SetupTestHeaders())
	test.OK(t, err)

	// Update patient case state to be active so that we can proceed forward as though the cases
	// were submitted
	case2, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv2.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, true, case2.Claimed)
	nextStatus := common.PCStatusActive
	test.OK(t, testData.DataAPI.UpdatePatientCase(case2.ID.Int64(), &api.PatientCaseUpdate{
		Status: &nextStatus,
	}))

	// doctor1 should be able to review the case with a selected doctor but not pick a TP for it
	dc1 := test_integration.DoctorClient(testData, t, dr.DoctorID)
	_, err = dc1.ReviewVisit(pv.PatientVisitID)
	test.OK(t, err)
	_, err = dc1.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)
	_, err = dc1.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.OK(t, err)
	_, err = dc1.PickTreatmentPlanForVisit(pv2.PatientVisitID, nil)
	test.Equals(t, true, err != nil)

	// doctor2 should not be able to pick a TP for it
	dc2 := test_integration.DoctorClient(testData, t, dr2.DoctorID)
	_, err = dc2.ReviewVisit(pv2.PatientVisitID)
	test.OK(t, err)
	_, err = dc2.PickTreatmentPlanForVisit(pv.PatientVisitID, nil)
	test.Equals(t, true, err != nil)
	_, err = dc2.PickTreatmentPlanForVisit(pv2.PatientVisitID, nil)
	test.OK(t, err)

}
