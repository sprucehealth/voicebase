package test_promotions

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_OnePromotionPerParkedAccount(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "5% off visit for new Spruce Users"
	successMsg := "Successfully claimed 5% coupon code"
	promoCode := createPromotion(promotions.NewPercentOffVisitPromotion(5,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// lets have a new user claim this code via the website
	done := make(chan bool, 1)
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	// give enough time for the promotion to get associated with the new user
	<-done
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataApi.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets have the same parked user claim another code
	promoCode2 := createPromotion(promotions.NewPercentOffVisitPromotion(5,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)
	_, err = promotions.AssociatePromoCode("kunal@test.com", "California", promoCode2, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	<-done

	// ensure that the parked account is still associated with the previous code
	parkedAccount, err = testData.DataApi.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

}

func TestPromotion_MoreMoneyThanCost(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	displayMsg := "$100 off visit for new Spruce Users"
	successMsg := "Successfully claimed $100 coupon code"
	promoCode := createPromotion(promotions.NewMoneyOffVisitPromotion(10000,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	pr := test_integration.SignupRandomTestPatient(t, testData)

	done := make(chan bool, 1)
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)
	<-done

	// query the cost of the visit to ensure that its not < 0
	cost, lineItems := queryCost(pr.Patient.AccountId.Int64(), testData, t)
	test.Equals(t, "$0", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, displayMsg, lineItems[1].Description)
	test.Equals(t, "-$40", lineItems[1].Value)
}

func TestPromotion_NonNewUser(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue

	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "5% off visit for new Spruce Users"
	successMsg := "Successfully claimed 5% coupon code"
	promoCode := createPromotion(promotions.NewPercentOffVisitPromotion(5,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	pv, _ := test_integration.CreateRandomPatientVisitAndPickTP(t, testData, doctor)
	patient, err := testData.DataApi.GetPatientFromPatientVisitId(pv.PatientVisitId)
	test.OK(t, err)

	// now try and claim the code for this user
	done := make(chan bool, 1)
	_, err = promotions.AssociatePromoCode(patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// now ensure that the user does not have a pending promotion in the account
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patient.AccountId.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))
}

func TestPromotion_SamePromotionCodeApplyAttempt(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "5% off visit for new Spruce Users"
	successMsg := "Successfully claimed 5% coupon code"
	promoCode := createPromotion(promotions.NewPercentOffVisitPromotion(5,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// create a user
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// get this patient to claim the code
	done := make(chan bool, 1)
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// now attempt to get this user to claim the code again
	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// there should only be 1 pending promotion in the user's acount
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(pr.Patient.AccountId.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))
}
