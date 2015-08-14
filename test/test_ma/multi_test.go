package test_ma

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPrimaryCC(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// Non primary CC should not be assigned to case

	test_integration.SignupRandomTestCC(t, testData, false)

	p := test_integration.SignupRandomTestPatient(t, testData)
	pcli := test_integration.PatientClient(testData, t, p.Patient.ID)
	res, err := pcli.CreatePatientVisit(api.AcnePathwayTag, 0, map[string][]string{
		"S-Version":   []string{"Patient;Feature;1.0.0;000105"},
		"S-OS":        []string{"iOS;7.1.1"},
		"S-Device":    []string{"Phone;iPhone6,1;640;1136;2.0"},
		"S-Device-ID": []string{"12345678-1234-1234-1234-123456789abc"},
	})
	test.OK(t, err)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(res.PatientVisitID)
	test.OK(t, err)
	assignments, err := testData.DataAPI.GetActiveMembersOfCareTeamForCase(patientVisit.PatientCaseID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 0, len(assignments))

	// A primary CC should be assigned to the case if available

	cc, _, _ := test_integration.SignupRandomTestCC(t, testData, true)

	p = test_integration.SignupRandomTestPatient(t, testData)
	pcli = test_integration.PatientClient(testData, t, p.Patient.ID)
	res, err = pcli.CreatePatientVisit(api.AcnePathwayTag, 0, map[string][]string{
		"S-Version":   []string{"Patient;Feature;1.0.0;000105"},
		"S-OS":        []string{"iOS;7.1.1"},
		"S-Device":    []string{"Phone;iPhone6,1;640;1136;2.0"},
		"S-Device-ID": []string{"12345678-1234-1234-1234-123456789abc"},
	})
	test.OK(t, err)
	patientVisit, err = testData.DataAPI.GetPatientVisitFromID(res.PatientVisitID)
	test.OK(t, err)
	assignments, err = testData.DataAPI.GetActiveMembersOfCareTeamForCase(patientVisit.PatientCaseID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(assignments))
	test.Equals(t, cc.DoctorID, assignments[0].ProviderID)
}
