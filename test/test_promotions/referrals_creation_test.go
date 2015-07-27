package test_promotions

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestReferrals_DoctorProgramCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// create a doctor and see if a referral program gets created for the doctor
	// with a deterministic code
	dr, email, password := test_integration.SignupRandomTestDoctor(t, testData)

	params := url.Values{}
	params.Set("email", email)
	params.Set("password", password)
	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.DoctorAuthenticateURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// at this point there should be a referral program for the doctor
	referralProgram, err := testData.DataAPI.ActiveReferralProgramForAccount(doctor.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, true, referralProgram != nil)

	// lets lookup the code by the expected referral code to see if it works
	displayInfo, err := promotions.LookupPromoCode(fmt.Sprintf("dr%s", doctor.LastName), testData.DataAPI, testData.Config.AnalyticsLogger)
	test.OK(t, err)
	test.Equals(t, true, displayInfo != nil)
	test.Equals(t, true, strings.Contains(displayInfo.Title, doctor.LastName))
}
