package test_doctor

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPacticeModelDoctorRegistration(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	pm, err := testData.DataAPI.PracticeModel(dr1.DoctorID)
	test.OK(t, err)
	test.Equals(t, pm.DoctorID, dr1.DoctorID)
	test.Equals(t, pm.IsSprucePC, true)
	test.Equals(t, pm.HasPracticeExtension, false)
}

func TestPacticeModelUpdate(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	dr1, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	pm, err := testData.DataAPI.PracticeModel(dr1.DoctorID)
	test.OK(t, err)
	test.Equals(t, pm.DoctorID, dr1.DoctorID)
	test.Equals(t, pm.IsSprucePC, true)
	test.Equals(t, pm.HasPracticeExtension, false)
	aff, err := testData.DataAPI.UpdatePracticeModel(dr1.DoctorID, &common.PracticeModelUpdate{
		IsSprucePC:           ptr.Bool(false),
		HasPracticeExtension: ptr.Bool(true),
	})
	test.OK(t, err)
	test.Equals(t, aff, int64(1))
	pm, err = testData.DataAPI.PracticeModel(dr1.DoctorID)
	test.OK(t, err)
	test.Equals(t, pm.DoctorID, dr1.DoctorID)
	test.Equals(t, pm.IsSprucePC, false)
	test.Equals(t, pm.HasPracticeExtension, true)
}
