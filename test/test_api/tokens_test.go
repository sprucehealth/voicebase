package test_api

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTokens(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	key, err := testData.DataAPI.ValidateToken("purpose", "token")
	test.Equals(t, api.ErrTokenDoesNotExist, err)
	test.Equals(t, "", key)

	token, err := testData.DataAPI.CreateToken("purpose", "key", "token", time.Hour)
	test.OK(t, err)
	test.Equals(t, "token", token)

	key, err = testData.DataAPI.ValidateToken("purpose", token)
	test.OK(t, err)
	test.Equals(t, "key", key)

	n, err := testData.DataAPI.DeleteToken("purpose", token)
	test.OK(t, err)
	test.Equals(t, 1, n)

	key, err = testData.DataAPI.ValidateToken("purpose", token)
	test.Equals(t, api.ErrTokenDoesNotExist, err)
	test.Equals(t, "", key)

	token, err = testData.DataAPI.CreateToken("purpose", "key", "token2", -time.Hour)
	test.OK(t, err)
	test.Equals(t, "token2", token)

	key, err = testData.DataAPI.ValidateToken("purpose", token)
	test.Equals(t, api.ErrTokenExpired, err)
	test.Equals(t, "", key)
}
