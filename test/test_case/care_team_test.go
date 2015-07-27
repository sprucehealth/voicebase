package test_case

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCareTeam_AddDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("PA", t, testData)

	pv := test_integration.CreateRandomPatientVisitInState("PA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pv.PatientVisitID)
	test.OK(t, err)
	// add the doctor to the patient case
	test.OK(t, testData.DataAPI.AddDoctorToPatientCase(dr.DoctorID, patientVisit.PatientCaseID.Int64()))

	// now try and add a doctor that is not registered to see patients in the patient's state but
	// is registered for the pathway
	dr2 := test_integration.SignupRandomTestDoctorInState("FL", t, testData)

	// the call should fail as the doctor is not registered to see patients in FL
	test.Equals(t, true, testData.DataAPI.AddDoctorToPatientCase(dr2.DoctorID, patientVisit.PatientCaseID.Int64()) != nil)

	// now create a different pathway for which to create a patient visit
	pathway := &common.Pathway{
		Name:           "test",
		Tag:            "test",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}
	err = testData.DataAPI.CreatePathway(pathway)
	test.OK(t, err)

	dr3 := test_integration.SignupRandomTestDoctorInState("NY", t, testData)

	// ensure that doctor3 is signed up for the new pathway (and NOT acne) only but in NY
	acnePathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)
	_, err = testData.DB.Exec(`
		DELETE cpse.* FROM care_provider_state_elligibility cpse
		INNER JOIN care_providing_state cps ON cps.id = cpse.care_providing_state_id
		WHERE cpse.provider_id = ?
		AND cps.clinical_pathway_id = ?`, dr.DoctorID, acnePathway.ID)
	test.OK(t, err)

	// now register the 3rd doctor for this pathway but in another state (NY)
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("NY", "New York", pathway.Tag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr3.DoctorID))

	// at this point try to add this doctor to the patient case (should fail because while the doctor is registered
	// for this particular pathway, the doctor is not reigstered in the patient's state)
	test.Equals(t, true, testData.DataAPI.AddDoctorToPatientCase(dr.DoctorID, patientVisit.PatientCaseID.Int64()) != nil)

}
