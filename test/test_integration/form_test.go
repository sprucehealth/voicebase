package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestForm(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	test.OK(t, testData.DataAPI.RecordForm(&common.NotifyMeForm{
		Email:     "test@test.com",
		State:     "CA",
		Platform:  "iOS",
		UniqueKey: "deviceID1234",
	}, "test", 12345))

	entryExists, err := testData.DataAPI.FormEntryExists("form_notify_me", "test@test.com")
	test.OK(t, err)
	test.Equals(t, false, entryExists)

	entryExists, err = testData.DataAPI.FormEntryExists("form_notify_me", "deviceID1234")
	test.OK(t, err)
	test.Equals(t, true, entryExists)

	// attempt to re-insert based on unique key
	test.OK(t, testData.DataAPI.RecordForm(&common.NotifyMeForm{
		Email:     "test@test.com",
		State:     "CA",
		Platform:  "iOS",
		UniqueKey: "deviceID1234",
	}, "test", 12345))

	// attempt to insert without specifing unique key
	test.OK(t, testData.DataAPI.RecordForm(&common.NotifyMeForm{
		Email:    "test@test.com",
		State:    "CA",
		Platform: "iOS",
	}, "test", 12345))

	test.OK(t, testData.DataAPI.RecordForm(&common.NotifyMeForm{
		Email:    "test@test.com",
		State:    "CA",
		Platform: "iOS",
	}, "test", 12345))

}
