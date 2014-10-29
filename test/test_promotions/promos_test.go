package test_promotions

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_NewUserPercentOff(t *testing.T) {
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

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)

	// wait for the promotion to be applied
	time.Sleep(300 * time.Millisecond)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataApi.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	patientAccountID := pr.Patient.AccountId.Int64()
	patientID := pr.Patient.PatientId.Int64()

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneVisit, testData, t)
	test.Equals(t, "$38", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, displayMsg, lineItems[1].Description)

	// lets make sure the pending promotion is reflected on the patient account
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))

	// now lets get this patient to submit a visit
	w, patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)
	defer w.Stop()

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 3800, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, displayMsg, patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the user has no more pending promotions
	pendingPromotions, err = testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))
}

func TestPromotion_ExistingUserPercentOff(t *testing.T) {
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

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// lets have this user claim the code
	done := make(chan bool, 1)
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// at this point there should be a pending promotion against the user's account
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(pr.Patient.AccountId.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))
}

func TestPromotion_NewUserDollarOff(t *testing.T) {
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
	displayMsg := "$25 off visit for new Spruce Users"
	successMsg := "Successfully claimed $25 coupon code"
	promoCode := createPromotion(promotions.NewMoneyOffVisitPromotion(2500,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// lets have a new user claim this code via the website
	done := make(chan bool, 1)
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	// wait for the promotion to get associated with the new user
	<-done
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// lets ensure that the parked account was created
	parkedAccount, err := testData.DataApi.ParkedAccount("kunal@test.com")
	test.OK(t, err)
	test.Equals(t, true, parkedAccount != nil)
	test.Equals(t, promoCode, parkedAccount.Code)

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)

	// wait for the promotion to be applied
	time.Sleep(300 * time.Millisecond)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataApi.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	patientAccountID := pr.Patient.AccountId.Int64()
	patientID := pr.Patient.PatientId.Int64()

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneVisit, testData, t)
	test.Equals(t, "$15", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, displayMsg, lineItems[1].Description)

	// lets make sure the pending promotion is reflected on the patient account
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))

	// now lets get this patient to submit a visit
	w, patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)
	defer w.Stop()

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 1500, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, displayMsg, patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the user has no more pending promotions
	pendingPromotions, err = testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))
}

func TestPromotion_ExistingUserDollarOff(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
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
		true), testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// lets have this user claim the code
	done := make(chan bool, 1)
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// at this point there should be a pending promotion against the user's account
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(pr.Patient.AccountId.Int64(), promotions.Types)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingPromotions))
}

func TestPromotion_NewUserAccountCredit(t *testing.T) {
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
	displayMsg := "$12 added to your account for new Spruce Users"
	successMsg := "Successfully claimed $12 coupon code"
	promoCode := createPromotion(promotions.NewAccountCreditPromotion(1200,
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

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)

	// wait for the promotion to be applied
	time.Sleep(300 * time.Millisecond)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataApi.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	patientAccountID := pr.Patient.AccountId.Int64()
	patientID := pr.Patient.PatientId.Int64()

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneVisit, testData, t)
	test.Equals(t, "$28", cost)
	test.Equals(t, 2, len(lineItems))
	test.Equals(t, "Spruce credits", lineItems[1].Description)

	// lets make sure there is no pending promotion given that we are applying account credit
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// there should be account credit in the patients account
	patientCredit, err := testData.DataApi.AccountCredit(patientAccountID)
	test.OK(t, err)
	test.Equals(t, 1200, patientCredit.Credit)

	// now lets get this patient to submit a visit
	w, patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)
	defer w.Stop()

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 2800, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 2, len(patientReciept.CostBreakdown.LineItems))
	test.Equals(t, "Spruce credits", patientReciept.CostBreakdown.LineItems[1].Description)

	// lets make sure the patient has no more account credit
	patientCredit, err = testData.DataApi.AccountCredit(patientAccountID)
	test.OK(t, err)
	test.Equals(t, 0, patientCredit.Credit)
}

func TestPromotion_ExistingUserAccountCredit(t *testing.T) {
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
	displayMsg := "$12 added to your account for new Spruce Users"
	successMsg := "Successfully claimed $12 coupon code"
	promoCode := createPromotion(promotions.NewAccountCreditPromotion(1200,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		true), testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// lets have this user claim the code
	done := make(chan bool, 1)
	_, err := promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// at this point there should be account credits in the user's account
	patientCredit, err := testData.DataApi.AccountCredit(pr.Patient.AccountId.Int64())
	test.OK(t, err)
	test.Equals(t, 1200, patientCredit.Credit)
}

func TestPromotion_NewUserRouteToDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visist
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorId,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		doctor.SmallThumbnailURL,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode := createPromotion(promotion, testData, t)

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

	// now lets create a patient with this email address
	pr := test_integration.SignupTestPatientWithEmail("kunal@test.com", t, testData)

	// wait for the promotion to be applied
	time.Sleep(300 * time.Millisecond)

	// ensure that the interstitial is shown to the patient
	test.Equals(t, true, pr.PromotionConfirmationContent != nil)
	test.Equals(t, successMessage, pr.PromotionConfirmationContent.BodyText)

	// lets make sure that parked account reflects that the patient was created
	parkedAccount, err = testData.DataApi.ParkedAccount(pr.Patient.Email)
	test.OK(t, err)
	test.Equals(t, true, parkedAccount.AccountCreated)

	patientAccountID := pr.Patient.AccountId.Int64()
	patientID := pr.Patient.PatientId.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $38
	cost, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneVisit, testData, t)
	test.Equals(t, "$40", cost)
	test.Equals(t, 1, len(lineItems))

	// lets make sure there is no pending promotion given that the promotion is specifically
	// to route a patient to a doctor
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// the doctor should already be part of the patient's care team
	careTeamMembers, err := testData.DataApi.GetActiveMembersOfCareTeamForPatient(patientID, false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorId, careTeamMembers[0].ProviderID)

	// now lets get this patient to submit a visit
	w, patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)
	defer w.Stop()

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 4000, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 1, len(patientReciept.CostBreakdown.LineItems))

	// lets make sure the visit lands into the queue of the doctor
	pendingItems, err := testData.DataApi.GetPendingItemsInDoctorQueue(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
	test.Equals(t, api.DQEventTypePatientVisit, pendingItems[0].EventType)
}

func TestPromotion_ExistingUserRouteToDoctor(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visist
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorId,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		doctor.SmallThumbnailURL,
		"new_user",
		displayMsg,
		displayMsg,
		successMsg,
		0,
		promotions.USDUnit)
	test.OK(t, err)
	promoCode := createPromotion(promotion, testData, t)

	// now lets make sure that an existing user can claim the code as well
	pr := test_integration.SignupRandomTestPatient(t, testData)

	// lets have this user claim the code
	done := make(chan bool, 1)
	_, err = promotions.AssociatePromoCode(pr.Patient.Email, "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	test.OK(t, err)
	<-done

	// at this point there should be a doctor part of the user's care team
	careTeamMembers, err := testData.DataApi.GetActiveMembersOfCareTeamForPatient(pr.Patient.PatientId.Int64(), false)
	test.OK(t, err)
	test.Equals(t, 1, len(careTeamMembers))
	test.Equals(t, dr.DoctorId, careTeamMembers[0].ProviderID)
}

// This test is to ensure that a patient that uses a route to doctor promotion
// does not blindly get routed to that doctor in the event the doctor is not licensed to see patients in that
// state
func TestPromotion_ExistingUserRouteToDoctor_Uneligible(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)
	setupPromotionsTest(testData, t)

	// lets create a doctor to which we'd like to route the visit
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)
	doctor, err := testData.DataApi.GetDoctorFromId(dr.DoctorId)
	test.OK(t, err)

	// create a percent off discount promotion
	displayMsg := "Get seen by a specific doctor"
	successMsg := "you will be routed to doctor"

	promotion, err := promotions.NewRouteDoctorPromotion(dr.DoctorId,
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		doctor.SmallThumbnailURL,
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

	// change the patient location to FL so that we can simulate the situation
	// where the patient enters from a state where the doctor is not eligible to see the
	_, err = testData.DB.Exec(`INSERT INTO care_providing_state (long_state, state, health_condition_id) values (?,?,?)`, "Florida", "FL", api.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	_, err = testData.DB.Exec(`UPDATE patient_location set state = ? where patient_id = ?`, "FL", pr.Patient.PatientId.Int64())

	// lets have a new user claim this code via the website
	done := make(chan bool, 1)
	successMessage, err := promotions.AssociatePromoCode("kunal@test.com", "California", promoCode, testData.DataApi, testData.AuthApi, testData.Config.AnalyticsLogger, done)
	// give enough time for the promotion to get associated with the new user
	<-done
	test.OK(t, err)
	test.Equals(t, true, successMessage != "")

	// wait for the promotion to be applied
	time.Sleep(300 * time.Millisecond)

	patientAccountID := pr.Patient.AccountId.Int64()
	patientID := pr.Patient.PatientId.Int64()
	test_integration.AddCreditCardForPatient(patientID, testData, t)

	// lets query the cost API to see what the patient would see for the cost of the visit
	// lets get the cost; it should be $40
	cost, lineItems := test_integration.QueryCost(patientAccountID, sku.AcneVisit, testData, t)
	test.Equals(t, "$40", cost)
	test.Equals(t, 1, len(lineItems))

	// lets make sure there is no pending promotion given that the promotion is specifically
	// to route a patient to a doctor
	pendingPromotions, err := testData.DataApi.PendingPromotionsForAccount(patientAccountID, promotions.Types)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingPromotions))

	// the doctor should not be part of the patient's care team
	careTeamMembers, err := testData.DataApi.GetActiveMembersOfCareTeamForPatient(patientID, false)
	test.OK(t, err)
	test.Equals(t, 0, len(careTeamMembers))

	// now lets get this patient to submit a visit
	w, patientVisitID := startAndSubmitVisit(patientID, patientAccountID, stubSQSQueue, testData, t)
	defer w.Stop()

	// lets ensure a receipt exists for the expected cost
	patientReciept := getPatientReceipt(patientID, patientVisitID, testData, t)
	test.Equals(t, 4000, patientReciept.CostBreakdown.TotalCost.Amount)
	test.Equals(t, 1, len(patientReciept.CostBreakdown.LineItems))

	// lets make sure the visit lands into the unassigned queue
	pendingItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 0, len(pendingItems))

	// ensure that the pending item is visible by a doctor that is ellgibile to see patients in FL
	drFL := test_integration.SignupRandomTestDoctorInState("FL", t, testData)
	pendingItems, err = testData.DataApi.GetElligibleItemsInUnclaimedQueue(drFL.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(pendingItems))
}
