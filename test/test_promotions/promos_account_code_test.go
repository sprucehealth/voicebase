package test_promotions

import (
	"strconv"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_LookupByAccountCode(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Create a referral program
	CreateReferralProgram(t, testData)

	// Create an account to map to this referral program
	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)

	// Validate that our patient hasn't had an account code generated yet
	code, err := testData.DataAPI.AccountCode(patient.AccountID.Int64())
	test.Assert(t, code == nil, "Expected account code to be nil")

	// Interact with the system the way a patient would who is going to view a referral link. This should map a referral program to the patient if they don't already have one
	rafDisplay, err := promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.test", patient.AccountID.Int64())
	test.OK(t, err)

	// Validate that our link contains our account code that is mapped to the patient
	code, err = testData.DataAPI.AccountCode(patient.AccountID.Int64())
	test.OK(t, err)
	test.Assert(t, code != nil, "Expected account code to no longer be nil")
	test.Assert(t, strings.Contains(rafDisplay.URL, strconv.FormatInt(int64(*code), 10)), "The patient's referral link should have contained the account code")

	// Validate that we can look up promo codes by account_code
	displayInfo, err := promotions.LookupPromoCode(strconv.FormatInt(int64(*code), 10), testData.DataAPI, testData.Config.AnalyticsLogger)
	test.OK(t, err)
	test.Equals(t, true, displayInfo != nil)
	test.Equals(t, "display_msg", displayInfo.Title)
}

func TestPromotion_AssociateRandomAccountCode_AccountForAccountCode(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Create an account to map to map our account code to
	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)

	// Validate that our patient hasn't had an account code generated yet
	code, err := testData.DataAPI.AccountCode(patient.AccountID.Int64())
	test.Assert(t, code == nil, "Expected account code to be nil")

	// Associate a random account code with our account
	associatedCode, err := testData.DataAPI.AssociateRandomAccountCode(patient.AccountID.Int64())
	test.OK(t, err)
	test.Assert(t, associatedCode != 0, "Expected non zero account code to be generated")

	// Lookup our account by the generated code
	account, err := testData.DataAPI.AccountForAccountCode(associatedCode)
	test.OK(t, err)
	test.Equals(t, account.ID, patient.AccountID.Int64())
}
