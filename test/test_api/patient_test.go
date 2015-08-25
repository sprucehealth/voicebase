package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestUnlinkedPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("d1@sprucehealth.com", "abc", api.RoleDoctor)
	if err != nil {
		t.Fatal(err)
	}
	d1 := &common.Doctor{
		AccountID: encoding.DeprecatedNewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterProvider(d1, api.RoleDoctor)
	test.OK(t, err)

	p := &common.Patient{
		FirstName:    "Test",
		LastName:     "TestLastName",
		DOB:          encoding.Date{Year: 2013, Month: 8, Day: 9},
		Email:        "test@test.com",
		Gender:       "male",
		ZipCode:      "90210",
		ERxPatientID: encoding.DeprecatedNewObjectID(12345),
		Pharmacy:     &pharmacy.PharmacyData{},
		PhoneNumbers: []*common.PhoneNumber{
			{
				Phone:  common.Phone("734-846-5522"),
				Type:   common.PNTCell,
				Status: api.StatusActive,
			},
			{
				Phone:  common.Phone("734-846-5523"),
				Type:   common.PNTCell,
				Status: api.StatusActive,
			},
		},
	}

	err = testData.DataAPI.CreateUnlinkedPatientFromRefillRequest(p, d1, api.AcnePathwayTag)
	if err != nil {
		t.Fatal(err)
	}

	retrievedPatient, err := testData.DataAPI.GetPatientFromID(p.ID)
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, p.FirstName, retrievedPatient.FirstName)
	test.Equals(t, p.LastName, retrievedPatient.LastName)
	test.Equals(t, p.DOB, retrievedPatient.DOB)
	test.Equals(t, p.Gender, retrievedPatient.Gender)
	test.Equals(t, p.ZipCode, retrievedPatient.ZipCode)
	test.Equals(t, p.ERxPatientID, retrievedPatient.ERxPatientID)
	test.Equals(t, p.PhoneNumbers[0].Phone, retrievedPatient.PhoneNumbers[0].Phone)

}
