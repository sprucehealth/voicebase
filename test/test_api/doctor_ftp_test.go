package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestDoctorSyncSFTPs(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("dr@sprucehealth.com", "abc", api.RoleDoctor)
	test.OK(t, err)

	doctor := &common.Doctor{
		AccountID: encoding.DeprecatedNewObjectID(accountID),
		Address:   &common.Address{},
	}
	_, err = testData.DataAPI.RegisterProvider(doctor, api.RoleDoctor)
	test.OK(t, err)

	// add global FTPs
	_, err = testData.DataAPI.InsertFavoriteTreatmentPlan(&common.FavoriteTreatmentPlan{
		Name:      "Test",
		Lifecycle: api.StatusActive,
	}, api.AcnePathwayTag, 0)
	if err != nil {
		t.Fatal(err)
	}

	// ensure there is 1 global FTP at this point
	globalFTPs, err := testData.DataAPI.GlobalFavoriteTreatmentPlans([]string{api.StatusActive})
	if err != nil {
		t.Fatal(err.Error())
	}
	test.Equals(t, 1, len(globalFTPs))

	// add FTP for doctor
	_, err = testData.DataAPI.InsertFavoriteTreatmentPlan(&common.FavoriteTreatmentPlan{
		Name:      "ForDoctor",
		CreatorID: ptr.Int64(doctor.ID.Int64()),
		Lifecycle: api.StatusActive,
	}, api.AcnePathwayTag, 0)
	if err != nil {
		t.Fatal(err)
	}

	ftps, err := testData.DataAPI.FavoriteTreatmentPlansForDoctor(doctor.ID.Int64(), "")
	if err != nil {
		t.Fatal(err)
	}

	// doctor should have 1 FTP
	test.Equals(t, 1, len(ftps[api.AcnePathwayTag]))

	if err := testData.DataAPI.SyncGlobalFTPsForDoctor(doctor.ID.Int64()); err != nil {
		t.Fatal(err)
	}

	// doctor should now have 2 FTPs
	ftps, err = testData.DataAPI.FavoriteTreatmentPlansForDoctor(doctor.ID.Int64(), "")
	if err != nil {
		t.Fatal(err)
	}

	// doctor should have 2 FTPs
	test.Equals(t, 2, len(ftps[api.AcnePathwayTag]))

}
