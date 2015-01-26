package test_promotions

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_GroupWithMultiplePromotions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a group where multiple promotions are allowed
	_, err := testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "convert",
		MaxAllowedPromos: 5,
	})

	// create a percent discount promotion
	displayMsg := "5% off visit for new Spruce Users"
	successMsg := "Successfully claimed 5% coupon code"
	promoCode1 := createPromotion(promotions.NewPercentOffVisitPromotion(5,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// create a dollar off discount promotion
	displayMsg = "$25 off visit for new Spruce Users"
	successMsg = "Successfully claimed $25 coupon code"
	promoCode2 := createPromotion(promotions.NewMoneyOffVisitPromotion(2500,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// lets create a route to doctor promotion
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a percent off discount promotion
	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorID,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		"thumbnail",
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode3 := createPromotion(promotion, testData, t)

	// create an account credits promotion
	promoCode4 := createPromotion(promotions.NewAccountCreditPromotion(1200,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// create another account credits promotion
	promoCode5 := createPromotion(promotions.NewAccountCreditPromotion(1000,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// now lets apply all these promotions to an existing patient's account
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode1, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode2, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode3, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode4, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode5, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger)
	// give enough time for the promotion to get associated with the new user
	test.OK(t, err)

	// at this point the patient should have $22 in credit
	patientCredit, err := testData.DataAPI.AccountCredit(pr.Patient.AccountID.Int64())
	test.OK(t, err)
	test.Equals(t, 2200, patientCredit.Credit)

	// at this point the patient should have 2 pending promotions
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(pr.Patient.AccountID.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, 2, len(pendingPromotions))

	// and the doctor added to their account
	careTeamMembers, err := testData.DataAPI.GetActiveMembersOfCareTeamForPatient(pr.Patient.PatientID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorID, careTeamMembers[0].ProviderID)

	// the cost of the visit should be $8 after the percent promotion and the account credits
	cost, lineItems := test_integration.QueryCost(pr.Patient.AccountID.Int64(), sku.AcneVisit, testData, t)
	test.Equals(t, "$16", cost)
	test.Equals(t, 3, len(lineItems))

}
