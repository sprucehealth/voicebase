package test_patient_visit

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// Test to ensure that the same user can start multiple visits
// for different pathways with the constraint being that just a single
// open case can exist per pathway
func TestPatientVisit_MultiplePathways(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create a doctor
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// create a patient
	pr := test_integration.SignupRandomTestPatient(t, testData)
	pc := test_integration.PatientClient(testData, t, pr.Patient.PatientID.Int64())

	// create new pathway
	p1 := &common.Pathway{
		Name:           "test1",
		MedicineBranch: "test",
		Tag:            "test1",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(p1))

	// create skus for the pathway
	s1 := &common.SKU{
		Type:         p1.Tag + "_" + common.SCVisit.String(),
		CategoryType: common.SCVisit,
	}
	_, err := testData.DataAPI.CreateSKU(s1)
	test.OK(t, err)

	// upload layouts for pathway
	test_integration.UploadLayoutPairForPathway(p1.Tag, testData, t)

	// register doctor in CA for this new pathway
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", p1.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr.DoctorID))

	// start a visit for the pathway
	_, err = pc.CreatePatientVisit(p1.Tag, dr.DoctorID, setupTestHeaders())
	test.OK(t, err)

	// create another pathway
	p2 := &common.Pathway{
		Name:           "test2",
		MedicineBranch: "test",
		Tag:            "test2",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(p2))

	// create sku for pathway
	s2 := &common.SKU{
		Type:         p2.Tag + "_" + common.SCVisit.String(),
		CategoryType: common.SCVisit,
	}
	_, err = testData.DataAPI.CreateSKU(s2)

	test.OK(t, err)
	// upload layouts for pathway
	test_integration.UploadLayoutPairForPathway(p2.Tag, testData, t)
	// register doctor in CA for this new pathway
	careProvidingStateID, err = testData.DataAPI.AddCareProvidingState("CA", "California", p2.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr.DoctorID))

	// start a visit for the pathway
	_, err = pc.CreatePatientVisit(p2.Tag, dr.DoctorID, setupTestHeaders())
	test.OK(t, err)

	// at this point the patient should have 2 open cases
	cases, err := testData.DataAPI.GetCasesForPatient(pr.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 2, len(cases))
	test.Equals(t, common.PCStatusUnclaimed, cases[0].Status)
	test.Equals(t, common.PCStatusUnclaimed, cases[1].Status)

}

func setupTestHeaders() http.Header {
	headers := http.Header(make(map[string][]string))
	headers.Set("S-Version", "Patient;Feature;1.0.0;000105")
	headers.Set("S-OS", "iOS;7.1.1")
	headers.Set("S-Device", "Phone;iPhone6,1;640;1136;2.0")
	headers.Set("S-Device-ID", "12345678-1234-1234-1234-123456789abc")
	return headers

}
