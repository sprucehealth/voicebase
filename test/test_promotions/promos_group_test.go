package test_promotions

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotionGroups(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DB.Exec(`INSERT INTO promotion_group (name, max_allowed_promos) VALUES ('attribution', 1)`)
	test.OK(t, err)
	_, err = testData.DB.Exec(`INSERT INTO promotion_group (name, max_allowed_promos) VALUES ('credit', 5)`)
	test.OK(t, err)

	groups, err := testData.DataAPI.PromotionGroups()
	test.OK(t, err)

	// Test that we get our two groups back
	test.Equals(t, 2, len(groups))

	// Test that we are returning them in a same ordering
	test.Equals(t, `attribution`, groups[0].Name)
	test.Equals(t, `credit`, groups[1].Name)
}

func TestPromotionGroup(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DB.Exec(`INSERT INTO promotion_group (name, max_allowed_promos) VALUES ('attribution', 1)`)
	test.OK(t, err)
	_, err = testData.DB.Exec(`INSERT INTO promotion_group (name, max_allowed_promos) VALUES ('credit', 5)`)
	test.OK(t, err)

	group, err := testData.DataAPI.PromotionGroup(`attribution`)
	test.OK(t, err)

	// Test that we are returning them in a same ordering
	test.Equals(t, `attribution`, group.Name)
	test.Equals(t, 1, group.MaxAllowedPromos)
	test.Assert(t, group.ID != 0, "Expected a non zero ID")
}

func TestCreatePromotionGroup(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "Foo",
		MaxAllowedPromos: 1,
	})

	group, err := testData.DataAPI.PromotionGroup(`Foo`)
	test.OK(t, err)

	// Test that we are returning them in a same ordering
	test.Equals(t, `Foo`, group.Name)
	test.Equals(t, 1, group.MaxAllowedPromos)
	test.Assert(t, group.ID != 0, "Expected a non zero ID")
}

func TestPromotion_GroupWithMultiplePromotions(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
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
	displayMsg := "50% off visit for new Spruce Users"
	successMsg := "Successfully claimed 50% coupon code"
	promoCode1 := createPromotion(promotions.NewPercentOffVisitPromotion(50,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// create a dollar off discount promotion
	displayMsg = "$5 off visit for new Spruce Users"
	successMsg = "Successfully claimed $5 coupon code"
	promoCode2 := createPromotion(promotions.NewMoneyOffVisitPromotion(500,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
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
	promoCode4 := createPromotion(promotions.NewAccountCreditPromotion(300,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// create another account credits promotion
	promoCode5 := createPromotion(promotions.NewAccountCreditPromotion(700,
		"convert",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// now lets apply all these promotions to an existing patient's account
	pr := signupPatientWithVisit("dagknag@gmail.com", testData, t)
	test_integration.AddTestAddressForPatient(pr.Patient.ID.Int64(), testData, t)
	test_integration.AddTestPharmacyForPatient(pr.Patient.ID.Int64(), testData, t)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode1, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, false)
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode2, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, false)
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode3, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, false)
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode4, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, false)
	test.OK(t, err)

	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode5, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, false)
	test.OK(t, err)

	// at this point the patient should have $10 in credit
	patientCredit, err := testData.DataAPI.AccountCredit(pr.Patient.AccountID.Int64())
	test.OK(t, err)
	test.Equals(t, 1000, patientCredit.Credit)

	// at this point the patient should have 2 pending promotions
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(pr.Patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 2, len(pendingPromotions))

	// and the doctor added to their account
	careTeamMembers, err := testData.DataAPI.GetActiveMembersOfCareTeamForPatient(pr.Patient.ID.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorID, careTeamMembers[0].ProviderID)

	// the cost of the visit should be $5 after the percent promotion, money promotion and the account credits
	cost, lineItems := test_integration.QueryCost(pr.Patient.AccountID.Int64(), test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$5", cost)
	test.Equals(t, 4, len(lineItems))
}
