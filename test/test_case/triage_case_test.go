package test_case

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that once a visit has been triaged pre-submission,
// the user can still create another case for the same pathway.
func TestPreSubmissionTriage(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.ID.Int64())

	pv, err := pc.CreatePatientVisit(api.AcnePathwayTag, 0, setupTestHeaders())
	test.OK(t, err)

	test.OK(t, pc.TriageVisit(pv.PatientVisitID))

	visit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusPreSubmissionTriage, visit.Status)
	pCase, err := testData.DataAPI.GetPatientCaseFromID(visit.PatientCaseID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PCStatusPreSubmissionTriage, pCase.Status)
	test.Equals(t, true, pCase.ClosedDate != nil)

	// now the patient should be able to start another visit
	pv2, err := pc.CreatePatientVisit(api.AcnePathwayTag, 0, setupTestHeaders())
	test.OK(t, err)
	test.Equals(t, true, pv.PatientVisitID != pv2.PatientVisitID)

	cases, err := testData.DataAPI.GetCasesForPatient(pr.Patient.ID.Int64(), nil)
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
}

func setupTestHeaders() http.Header {
	headers := http.Header(make(map[string][]string))
	headers.Set("S-Version", "Patient;Feature;1.0.0;000105")
	headers.Set("S-OS", "iOS;7.1.1")
	headers.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	headers.Set("S-Device-ID", "12345678-1234-1234-1234-123456789abc")
	return headers

}
