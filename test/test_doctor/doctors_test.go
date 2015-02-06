package test_doctor

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

// This test is to ensure that doing a query for
// multiple doctors works as expected
func TestDoctorsMultiQuery(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr3, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// query for all doctors
	doctors, err := testData.DataAPI.Doctors([]int64{dr1.DoctorID, dr2.DoctorID, 100, dr3.DoctorID})
	test.OK(t, err)
	test.Equals(t, 4, len(doctors))
	test.Equals(t, dr1.DoctorID, doctors[0].DoctorID.Int64())
	test.Equals(t, dr2.DoctorID, doctors[1].DoctorID.Int64())
	test.Equals(t, true, doctors[2] == nil)
	test.Equals(t, dr3.DoctorID, doctors[3].DoctorID.Int64())
}

// This test is to ensure that only eligible and available
// doctors are returned when querying for doctors eligible
// for a given pathway/state combination
func TestDoctors_AvailableForCareProvidingState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr3, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// create pathway
	p1 := &common.Pathway{
		Name:           "test1",
		Tag:            "test1",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}

	test.OK(t, testData.DataAPI.CreatePathway(p1))

	// register this new pathway as pathway to support on Spruce in FL
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("FL", "Floriday", p1.Tag)
	test.OK(t, err)

	// add all 3 doctors as eligible for the pathway/state combination
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr1.DoctorID))
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr2.DoctorID))
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr3.DoctorID))

	// ensure that we get all three doctors back when querying for available and eligible doctors
	doctorIDs, err := testData.DataAPI.DoctorIDsInCareProvidingState(careProvidingStateID)
	test.OK(t, err)
	test.Equals(t, 3, len(doctorIDs))

	eligibleDoctorIDs := make(map[int64]bool)
	for _, doctorID := range doctorIDs {
		eligibleDoctorIDs[doctorID] = true
	}

	// ensure that the doctors returned were the ones that were expected
	for _, doctorID := range []int64{dr1.DoctorID, dr2.DoctorID, dr3.DoctorID} {
		test.Equals(t, true, eligibleDoctorIDs[doctorID])
		eligibleDoctorIDs[doctorID] = false
	}

	// now make one of the doctors unavailable
	_, err = testData.DB.Exec(`
		UPDATE care_provider_state_elligibility
		SET unavailable = 1
		WHERE provider_id = ?`, dr2.DoctorID)
	test.OK(t, err)

	// now ensure only 2 doctors are returned
	doctorIDs, err = testData.DataAPI.DoctorIDsInCareProvidingState(careProvidingStateID)
	test.OK(t, err)
	test.Equals(t, 2, len(doctorIDs))
}

// This test is to ensure that given a set of doctorIDs and careProvidingStateID
// we are able to determine which doctors are eligible and available in the provided
// careProvidingStateID
func TestDoctors_EligibleQuery(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr2, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	dr3, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// create pathway
	p1 := &common.Pathway{
		Name:           "test1",
		Tag:            "test1",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}

	test.OK(t, testData.DataAPI.CreatePathway(p1))

	// register this new pathway as pathway to support on Spruce in FL
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("FL", "Floriday", p1.Tag)
	test.OK(t, err)

	// add 2 doctors as eligible for the pathway/state combination
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr1.DoctorID))
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr2.DoctorID))

	// provide a list of 3 doctors to see which ones are returned as being eligible
	eligibleDoctorIDs, err := testData.DataAPI.EligibleDoctorIDs([]int64{dr1.DoctorID, dr2.DoctorID, dr3.DoctorID}, careProvidingStateID)
	test.OK(t, err)
	// only 2 of the doctors should be returned as being eligible
	test.Equals(t, 2, len(eligibleDoctorIDs))
	// dr3 should not be included in this list
	test.Equals(t, true, eligibleDoctorIDs[0] != dr3.DoctorID)
	test.Equals(t, true, eligibleDoctorIDs[1] != dr3.DoctorID)

	// now make dr2 unavaiable
	_, err = testData.DB.Exec(`
		UPDATE care_provider_state_elligibility
		SET unavailable = 1
		WHERE provider_id = ?`, dr2.DoctorID)
	test.OK(t, err)

	// now only one doctor should be returned as being eligible
	eligibleDoctorIDs, err = testData.DataAPI.EligibleDoctorIDs([]int64{dr1.DoctorID, dr2.DoctorID, dr3.DoctorID}, careProvidingStateID)
	test.OK(t, err)
	test.Equals(t, 1, len(eligibleDoctorIDs))
	test.Equals(t, dr1.DoctorID, eligibleDoctorIDs[0])

}

// This test is to ensure that we are able to list available
// doctorIDs
func TestDoctors_ListAvailableDoctors(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	// remove all doctor assignments to make test deterministic
	_, err := testData.DB.Exec(`DELETE FROM care_provider_state_elligibility`)
	test.OK(t, err)

	// get same doctor registered in multiple states
	careProvidingStateID, err := testData.DataAPI.AddCareProvidingState("FL", "Florida", api.AcnePathwayTag)
	test.OK(t, err)

	careProvidingStateID2, err := testData.DataAPI.AddCareProvidingState("NY", "New York", api.AcnePathwayTag)
	test.OK(t, err)

	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID, dr.DoctorID))
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(careProvidingStateID2, dr.DoctorID))

	doctorIDs, err := testData.DataAPI.AvailableDoctorIDs(3)
	test.OK(t, err)
	test.Equals(t, 1, len(doctorIDs))
}
