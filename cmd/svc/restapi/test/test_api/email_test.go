package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/test/test_integration"
	"github.com/sprucehealth/backend/libs/test"
)

func TestEmailCampaignState(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	data, err := testData.DataAPI.EmailCampaignState("camp")
	test.OK(t, err)
	test.Equals(t, []byte(nil), data)

	test.OK(t, testData.DataAPI.UpdateEmailCampaignState("camp", []byte("foo")))

	data, err = testData.DataAPI.EmailCampaignState("camp")
	test.OK(t, err)
	test.Equals(t, []byte("foo"), data)
}