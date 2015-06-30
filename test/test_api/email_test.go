package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestEmailCampaignState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	data, err := testData.DataAPI.EmailCampaignState("camp")
	test.OK(t, err)
	test.Equals(t, []byte(nil), data)

	test.OK(t, testData.DataAPI.UpdateEmailCampaignState("camp", []byte("foo")))

	data, err = testData.DataAPI.EmailCampaignState("camp")
	test.OK(t, err)
	test.Equals(t, []byte("foo"), data)
}
