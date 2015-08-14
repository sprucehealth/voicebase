package test_doctor

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseList_PreSubmissionTriaged(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a case that gets submitted to a doctor
	p1 := test_integration.CreatePathway(t, testData, "pathway")
	pr := test_integration.SignupRandomTestPatient(t, testData)
	patient, err := testData.DataAPI.GetPatientFromID(pr.Patient.ID)
	test.OK(t, err)
	_, tp := test_integration.CreateRandomPatientVisitAndPickTPForPathway(t, testData, p1, patient, doctor)

	pc := test_integration.PatientClient(testData, t, tp.PatientID)

	// create another case that gets triaged pre-submission
	pv, err := pc.CreatePatientVisit(api.AcnePathwayTag, 0, test_integration.SetupTestHeaders())
	test.OK(t, err)

	test.OK(t, pc.TriageVisit(pv.PatientVisitID))

	// only a single case should be returned to the doctor (not the triaged one)
	dc := test_integration.DoctorClient(testData, t, dr.DoctorID)
	cases, err := dc.CasesForPatient(pr.Patient.ID)
	test.OK(t, err)
	test.Equals(t, 1, len(cases))
	test.Equals(t, cases[0].ID, tp.PatientCaseID.Int64())
}
