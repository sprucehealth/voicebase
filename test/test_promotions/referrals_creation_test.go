package test_promotions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestReferrals_DoctorProgramCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// create a doctor and see if a referral program gets created for the doctor
	// with a deterministic code
	dr, email, password := test_integration.SignupRandomTestDoctor(t, testData)

	params := url.Values{}
	params.Set("email", email)
	params.Set("password", password)
	req, err := http.NewRequest("POST", testData.APIServer.URL+router.DoctorAuthenticateURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// given that the referral program is created asynchronously wait for a moment
	time.Sleep(500 * time.Millisecond)
	// at this point there should be a referral program for the doctor
	referralProgram, err := testData.DataApi.ActiveReferralProgramForAccount(doctor.AccountId.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, true, referralProgram != nil)

	// lets lookup the code by the expected referral code to see if it works
	displayInfo, err := promotions.LookupPromoCode(fmt.Sprintf("dr%s", doctor.LastName), testData.DataApi, testData.Config.AnalyticsLogger)
	test.OK(t, err)
	test.Equals(t, true, displayInfo != nil)
	test.Equals(t, true, strings.Contains(displayInfo.Title, doctor.LastName))
}

func TestReferrals_PatientProgramCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)

	// create referral program template
	title := "pecentage off"
	description := "description"
	requestData := map[string]interface{}{
		"type":        "promo_money_off",
		"title":       title,
		"description": description,
		"group":       "new_user",
		"promotion": map[string]interface{}{
			"display_msg":  "percent off",
			"success_msg":  "percent off",
			"short_msg":    "percent off",
			"for_new_user": true,
			"group":        "new_user",
			"value":        50,
		},
	}

	var responseData map[string]interface{}
	resp, err := testData.AuthPostJSON(testData.APIServer.URL+router.ReferralProgramsTemplateURLPath, admin.AccountId.Int64(), requestData, &responseData)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now create patient
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// now try to get the referral program for this patient
	resp, err = testData.AuthGet(testData.APIServer.URL+router.ReferralsURLPath, pr.Patient.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, title, responseData["title"].(string))
	test.Equals(t, description, responseData["body_text"].(string))
	test.Equals(t, true, responseData["url"].(string) != "")

	// now update the referral program template
	newDescription := "new description"
	newTitle := "new title"
	requestData["title"] = newTitle
	requestData["description"] = newDescription

	resp, err = testData.AuthPostJSON(testData.APIServer.URL+router.ReferralProgramsTemplateURLPath, admin.AccountId.Int64(), requestData, &responseData)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now when we get the referral program for the patient it should reflect the new program
	// now try to get the referral program for this patient
	resp, err = testData.AuthGet(testData.APIServer.URL+router.ReferralsURLPath, pr.Patient.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	err = json.NewDecoder(resp.Body).Decode(&responseData)
	test.OK(t, err)
	test.Equals(t, newTitle, responseData["title"].(string))
	test.Equals(t, newDescription, responseData["body_text"].(string))
	test.Equals(t, true, responseData["url"].(string) != "")
}
