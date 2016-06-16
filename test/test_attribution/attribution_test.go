package test_attribution

import (
	"testing"

	"github.com/sprucehealth/backend/attribution/model"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestAttributionDeviceIDRecord(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	deviceID := ptr.String("Crazy-Device-ID")
	_, err := testData.DataAPI.InsertAttributionData(&model.AttributionData{DeviceID: deviceID, Data: map[string]interface{}{"Foo": "Bar"}})
	test.OK(t, err)
	ad, err := testData.DataAPI.LatestDeviceAttributionData(*deviceID)
	test.OK(t, err)
	fooValue, ok := ad.Data["Foo"]
	test.Equals(t, true, ok)
	foo, ok := fooValue.(string)
	test.Equals(t, true, ok)
	test.Equals(t, "Bar", foo)
	_, err = testData.DataAPI.InsertAttributionData(&model.AttributionData{DeviceID: deviceID, Data: map[string]interface{}{"Foo": "Baz"}})
	test.OK(t, err)
	ad, err = testData.DataAPI.LatestDeviceAttributionData(*deviceID)
	test.OK(t, err)
	fooValue, ok = ad.Data["Foo"]
	test.Equals(t, true, ok)
	foo, ok = fooValue.(string)
	test.Equals(t, true, ok)
	test.Equals(t, "Baz", foo)
}

func TestAttributionAccountIDRecord(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	pr := test_integration.SignupRandomTestPatientInState("CA", t, testData)
	accountID := pr.Patient.AccountID.Int64()
	_, err := testData.DataAPI.InsertAttributionData(&model.AttributionData{AccountID: ptr.Int64(accountID), Data: map[string]interface{}{"Foo": "Bar"}})
	test.OK(t, err)
	ad, err := testData.DataAPI.LatestAccountAttributionData(accountID)
	test.OK(t, err)
	fooValue, ok := ad.Data["Foo"]
	test.Equals(t, true, ok)
	foo, ok := fooValue.(string)
	test.Equals(t, true, ok)
	test.Equals(t, "Bar", foo)
	_, err = testData.DataAPI.InsertAttributionData(&model.AttributionData{AccountID: ptr.Int64(accountID), Data: map[string]interface{}{"Foo": "Baz"}})
	test.OK(t, err)
	ad, err = testData.DataAPI.LatestAccountAttributionData(accountID)
	test.OK(t, err)
	fooValue, ok = ad.Data["Foo"]
	test.Equals(t, true, ok)
	foo, ok = fooValue.(string)
	test.Equals(t, true, ok)
	test.Equals(t, "Baz", foo)
}

func TestAttributionDeleteRecord(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	deviceID := ptr.String("Crazy-Device-ID")
	_, err := testData.DataAPI.InsertAttributionData(&model.AttributionData{DeviceID: deviceID, Data: map[string]interface{}{"Foo": "Bar"}})
	test.OK(t, err)
	_, err = testData.DataAPI.LatestDeviceAttributionData(*deviceID)
	test.OK(t, err)
	aff, err := testData.DataAPI.DeleteAttributionData(*deviceID)
	test.OK(t, err)
	test.Equals(t, int64(1), aff)
	_, err = testData.DataAPI.LatestDeviceAttributionData(*deviceID)
	test.Assert(t, api.IsErrNotFound(err), "Expected no record to be found")
}
