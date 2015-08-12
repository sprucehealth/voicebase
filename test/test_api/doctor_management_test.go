package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestAvailableStates(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	// No doctors registered sanity check
	states, err := testData.DataAPI.AvailableStates()
	test.OK(t, err)
	test.Equals(t, 0, len(states))

	accountID, err := testData.AuthAPI.CreateAccount("dr@sprucehealth.com", "abc", api.RoleDoctor)
	test.OK(t, err)

	doctor := &common.Doctor{
		AccountID: encoding.NewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterProvider(doctor, api.RoleDoctor)
	test.OK(t, err)

	// Doctor registered but not elligible in any state
	states, err = testData.DataAPI.AvailableStates()
	test.OK(t, err)
	test.Equals(t, 0, len(states))

	cpStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", api.AcnePathwayTag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(cpStateID, doctor.ID.Int64()))

	states, err = testData.DataAPI.AvailableStates()
	test.OK(t, err)
	test.Equals(t, 1, len(states))
	test.Equals(t, "CA", states[0].Abbreviation)
	test.Equals(t, "California", states[0].Name)
}

func TestCareProviderEligible(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("dr@sprucehealth.com", "abc", api.RoleDoctor)
	test.OK(t, err)

	doctor := &common.Doctor{
		AccountID: encoding.NewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterProvider(doctor, api.RoleDoctor)
	test.OK(t, err)

	eligible, err := testData.DataAPI.CareProviderEligible(doctor.ID.Int64(), api.RoleDoctor, "CA", api.AcnePathwayTag)
	test.OK(t, err)
	test.Equals(t, false, eligible)

	// register doctor for acne in CA
	cpStateID, err := testData.DataAPI.AddCareProvidingState("CA", "California", api.AcnePathwayTag)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.MakeDoctorElligibleinCareProvidingState(cpStateID, doctor.ID.Int64()))

	eligible, err = testData.DataAPI.CareProviderEligible(doctor.ID.Int64(), api.RoleDoctor, "CA", api.AcnePathwayTag)
	test.OK(t, err)
	test.Equals(t, true, eligible)

}
