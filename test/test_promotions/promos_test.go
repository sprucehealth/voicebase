package test_promotions

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_NewUserPercentOff(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
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
		"",
		0,
		0,
		true), testData, t)

	// lets have a new user claim this code via the website
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataAPI.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)
	patientAccountID := pr.Patient.AccountID.Int64()
	patientID := pr.Patient.ID

	test_integration.AddTestPharmacyForPatient(patientID, testData, t)
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	test_integration.AddTestAddressForPatient(patientID, testData, t)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataAPI.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$38", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, displayMsg, lineItems[1].Description)

	// lets make sure the pending promotion is reflected on the patient account
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))

	// now lets get this patient to submit a visit
	patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 3800, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, displayMsg, patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the user has no more pending promotions
	pendingPromotions, err = testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))
}

func TestPromotion_ExistingUserPercentOff(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
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
		"",
		0,
		0,
		true), testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	// lets have this user claim the code
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)

	// at this point there should be a pending promotion against the user's account
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(pr.Patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))
}

func TestPromotion_NewUserDollarOff(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "$25 off visit for new Spruce Users"
	successMsg := "Successfully claimed $25 coupon code"
	promoCode := createPromotion(promotions.NewMoneyOffVisitPromotion(2500,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// lets have a new user claim this code via the website
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataAPI.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)
	patientAccountID := pr.Patient.AccountID.Int64()
	patientID := pr.Patient.ID
	test_integration.AddTestPharmacyForPatient(patientID, testData, t)
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	test_integration.AddTestAddressForPatient(patientID, testData, t)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataAPI.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$15", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, displayMsg, lineItems[1].Description)

	// lets make sure the pending promotion is reflected on the patient account
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))

	// now lets get this patient to submit a visit
	patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 1500, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, displayMsg, patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the user has no more pending promotions
	pendingPromotions, err = testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))
}

func TestPromotion_ExistingUserDollarOff(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "$25 off visit for new Spruce Users"
	successMsg := "Successfully claimed $25 coupon code"
	promoCode := createPromotion(promotions.NewMoneyOffVisitPromotion(2500,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	// lets have this user claim the code
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)

	// at this point there should be a pending promotion against the user's account
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(pr.Patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))
}

func TestPromotion_NewUserAccountCredit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "$12 added to your account for new Spruce Users"
	successMsg := "Successfully claimed $12 coupon code"
	promoCode := createPromotion(promotions.NewAccountCreditPromotion(1200,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// lets have a new user claim this code via the website
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataAPI.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)
	patientAccountID := pr.Patient.AccountID.Int64()
	patientID := pr.Patient.ID
	test_integration.AddTestPharmacyForPatient(patientID, testData, t)
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	test_integration.AddTestAddressForPatient(patientID, testData, t)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataAPI.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$28", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, "Credits", lineItems[1].Description)

	// lets make sure there is no pending promotion given that we are applying account credit
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// there should be account credit in the patients account
	patientCredit, err := testData.DataAPI.AccountCredit(patientAccountID)
	test.OK(t, err)
	test.Equals(t, 1200, patientCredit.Credit)

	// now lets get this patient to submit a visit
	patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 2800, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, "Credits", patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the patient has no more account credit
	patientCredit, err = testData.DataAPI.AccountCredit(patientAccountID)
	test.OK(t, err)
	test.Equals(t, 0, patientCredit.Credit)
}

func TestPromotion_ExistingUserAccountCredit(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// create a percent off discount promotion
	displayMsg := "$12 added to your account for new Spruce Users"
	successMsg := "Successfully claimed $12 coupon code"
	promoCode := createPromotion(promotions.NewAccountCreditPromotion(1200,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		"",
		0,
		0,
		true), testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatientWithPharmacyAndAddress(t, testData)

	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)

	// at this point there should be account credits in the user's account
	patientCredit, err := testData.DataAPI.AccountCredit(pr.Patient.AccountID.Int64())
	test.OK(t, err)
	test.Equals(t, 1200, patientCredit.Credit)
}

func TestPromotion_NewUserRouteToDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visist
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorID,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		"thumbnail",
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode := createPromotion(promotion, testData, t)

	// lets have a new user claim this code via the website
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataAPI.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets create a patient with this email address
	pr := signupPatientWithVisit("kunal@test.com", testData, t)
	patientAccountID := pr.Patient.AccountID.Int64()
	patientID := pr.Patient.ID
	test_integration.AddTestPharmacyForPatient(patientID, testData, t)
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	test_integration.AddTestAddressForPatient(patientID, testData, t)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataAPI.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$40", cost)
	test.Equals(t, 1, len(lineItems))

	// lets make sure there is no pending promotion given that the promotion is specifically
	// to route a patient to a doctor
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// the doctor should already be part of the patient's care team
	careTeamMembers, err := testData.DataAPI.GetActiveMembersOfCareTeamForPatient(patientID, false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorID, careTeamMembers[0].ProviderID)

	// now lets get this patient to submit a visit
	patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 4000, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 1, len(patientReciept.CostBreakdown.LineItems))

	// lets make sure the visit lands into the queue of the doctor
	pendingItems, err := testData.DataAPI.GetPendingItemsInDoctorQueue(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypePatientVisit, pendingItems[0].EventType)
}

func TestPromotion_ExistingUserRouteToDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visist
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorID,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		"thumbnail",
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode := createPromotion(promotion, testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := signupPatientWithVisit("Gdgkngng@gmail.com", testData, t)
	test_integration.AddTestAddressForPatient(pr.Patient.ID, testData, t)
	test_integration.AddTestPharmacyForPatient(pr.Patient.ID, testData, t)

	// lets have this user claim the code
	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)

	// at this point there should be a doctor part of the user's care team
	careTeamMembers, err := testData.DataAPI.GetActiveMembersOfCareTeamForPatient(pr.Patient.ID, false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorID, careTeamMembers[0].ProviderID)
}

// This test is to ensure that a patient that uses a route to doctor promotion
// does not blindly get routed to that doctor in the event the doctor is not licensed to see patients in that
// state
func TestPromotion_ExistingUserRouteToDoctor_Uneligible(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &awsutil.SQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visit
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorID,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		"thumbnail",
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode := createPromotion(promotion, testData, t)

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)
	patientAccountID := pr.Patient.AccountID.Int64()
	patientID := pr.Patient.ID
	test_integration.AddTestPharmacyForPatient(patientID, testData, t)
	test_integration.AddCreditCardForPatient(patientID, testData, t)
	test_integration.AddTestAddressForPatient(patientID, testData, t)

	pathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)

	// change the patient location to FL so that we can simulate the situation
	// where the patient enters from a state where the doctor is not eligible to see the
	_, err = testData.DB.Exec(`INSERT INTO care_providing_state (long_state, state, clinical_pathway_id) values (?,?,?)`, "Florida", "FL",
		pathway.ID)
	test.OK(t, err)
	_, err = testData.DB.Exec(`UPDATE patient_location set state = ? where patient_id = ?`, "FL", pr.Patient.ID.Int64())

	// lets have a new user claim this code via the website
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataAPI, testData.AuthAPI, testData.Config.AnalyticsLogger, true)
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $40
	cost, lineItems := test_integration.QueryCost(patientAccountID, test_integration.SKUAcneVisit, testData, t)
	test.Equals(t, "$40", cost)
	test.Equals(t, 1, len(lineItems))

	// lets make sure there is no pending promotion given that the promotion is specifically
	// to route a patient to a doctor
	pendingPromotions, err := testData.DataAPI.PendingPromotionsForAccount(patientAccountID, common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// the doctor should not be part of the patient's care team
	careTeamMembers, err := testData.DataAPI.GetActiveMembersOfCareTeamForPatient(patientID, false)
	test.OK(t, err)
	test.Equals(t, 0, len(careTeamMembers))

	// now lets get this patient to submit a visit
	patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 4000, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 1, len(patientReciept.CostBreakdown.LineItems))

	// lets make sure the visit lands into the unassigned queue
	pendingItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	// ensure that the pending item is visible by a doctor that is ellgibile to see patients in FL
	drFL := test_integration.SignupRandomTestDoctorInState("FL", t, testData)
	pendingItems, err = testData.DataAPI.GetElligibleItemsInUnclaimedQueue(drFL.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
}
