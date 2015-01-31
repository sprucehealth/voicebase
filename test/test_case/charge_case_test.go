package test_case

import (
	"errors"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestSucessfulCaseCharge(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, card := test_integration.SetupTestWithActiveCostAndVisitSubmitted(testData, t)

	// now lets go ahead and start the work to consume the message
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	w.Do()

	// at this point there should be a patient receipt, with a stripe charge and a credit card set, the status should be email sent
	patientReceipt, err := testData.DataAPI.GetPatientReceipt(patientVisit.PatientID.Int64(), patientVisit.PatientVisitID.Int64(), patientVisit.SKUType, true)
	test.OK(t, err)
	test.Equals(t, true, patientReceipt != nil)
	test.Equals(t, true, patientReceipt.CreditCardID == card.ID.Int64())
	test.Equals(t, "charge_test", patientReceipt.StripeChargeID)
	test.Equals(t, common.PRCharged, patientReceipt.Status)
	test.Equals(t, 1, len(patientReceipt.CostBreakdown.LineItems))

	// patient visit should indicate that the message was routed
	patientVisit, err = testData.DataAPI.GetPatientVisitFromID(patientVisit.PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)

	// there should be a pending item in the unclaimed queue
	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	eligibleItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(dr.DoctorID)
	test.OK(t, err)
	test.Equals(t, 1, len(eligibleItems))
}

func TestSuccessfulCharge_AlreadyExists(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, _ := test_integration.SetupTestWithActiveCostAndVisitSubmitted(testData, t)

	itemCost, err := testData.DataAPI.GetActiveItemCost(patientVisit.SKUType)
	test.OK(t, err)

	// lets create a receipt and have it exist in a state where its already in the end state
	patientReceipt := &common.PatientReceipt{
		ReferenceNumber: "12345",
		SKUType:         patientVisit.SKUType,
		ItemID:          patientVisit.PatientVisitID.Int64(),
		PatientID:       patientVisit.PatientID.Int64(),
		Status:          common.PRCharged,
		CostBreakdown:   &common.CostBreakdown{},
		ItemCostID:      itemCost.ID,
	}
	err = testData.DataAPI.CreatePatientReceipt(patientReceipt)
	test.OK(t, err)

	// lets ensure that there is no charge made again
	var wasChargeMade bool
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		wasChargeMade = true
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	w.Do()

	// lets make sure no charge was made and that just one patient receipt exists
	test.Equals(t, false, wasChargeMade)
	var count int
	err = testData.DB.QueryRow(`select count(*) from patient_receipt where patient_id = ?`, patientVisit.PatientID.Int64()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)
	patientReceipt, err = testData.DataAPI.GetPatientReceipt(patientVisit.PatientID.Int64(), patientVisit.PatientVisitID.Int64(), patientVisit.SKUType, true)
	test.OK(t, err)
	test.Equals(t, common.PRCharged, patientReceipt.Status)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataAPI.GetPatientVisitFromID(patientVisit.PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}
func TestFailedCharge_StripeFailure(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, card := test_integration.SetupTestWithActiveCostAndVisitSubmitted(testData, t)

	// lets fail the charge the first time to ensure that message doesn't get routed
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return nil, errors.New("charge error")
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	w.Do()

	// at this point the patient receipt should indicate that a charge is still pending
	patientReceipt, err := testData.DataAPI.GetPatientReceipt(patientVisit.PatientID.Int64(), patientVisit.PatientVisitID.Int64(), patientVisit.SKUType, false)
	test.OK(t, err)
	test.Equals(t, common.PRChargePending, patientReceipt.Status)
	test.Equals(t, int64(0), patientReceipt.CreditCardID)
	test.Equals(t, "", patientReceipt.StripeChargeID)

	// now lets get the charge to go through and it should succeed
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}
	w.Do()

	// at this point the charge should go through and there should be just 1 patient receipt existing for the patient
	var count int
	err = testData.DB.QueryRow(`select count(*) from patient_receipt where patient_id = ?`, patientVisit.PatientID.Int64()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)
	patientReceipt, err = testData.DataAPI.GetPatientReceipt(patientVisit.PatientID.Int64(), patientVisit.PatientVisitID.Int64(), patientVisit.SKUType, true)
	test.OK(t, err)
	test.Equals(t, common.PRCharged, patientReceipt.Status)
	test.Equals(t, card.ID.Int64(), patientReceipt.CreditCardID)
	test.Equals(t, "charge_test", patientReceipt.StripeChargeID)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataAPI.GetPatientVisitFromID(patientVisit.PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}

func TestFailedCharge_ChargeExists(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, _ := test_integration.SetupTestWithActiveCostAndVisitSubmitted(testData, t)

	itemCost, err := testData.DataAPI.GetActiveItemCost(patientVisit.SKUType)
	test.OK(t, err)

	// lets create a receipt and have it already exist to simulate a situation where a charge was started but failed for some reason
	patientReceipt := &common.PatientReceipt{
		ReferenceNumber: "12345",
		SKUType:         patientVisit.SKUType,
		ItemID:          patientVisit.PatientVisitID.Int64(),
		PatientID:       patientVisit.PatientID.Int64(),
		Status:          common.PRChargePending,
		CostBreakdown:   &common.CostBreakdown{},
		ItemCostID:      itemCost.ID,
	}
	err = testData.DataAPI.CreatePatientReceipt(patientReceipt)
	test.OK(t, err)

	// now lets get the charge to exist on stripe but not in our system, and lets keep track of
	// whether or not there is still an attempt to charge the customer
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.ListAllChargesFunc = func(string) ([]*stripe.Charge, error) {
		return []*stripe.Charge{
			&stripe.Charge{
				ID: "charge_test1234",
				Metadata: map[string]string{
					"receipt_ref_num": "12345",
				},
				Card: &stripe.Card{
					ID: "tpid3",
				},
			},
		}, nil
	}
	var wasCustomerCharged bool
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		wasCustomerCharged = true
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	w.Do()

	test.Equals(t, false, wasCustomerCharged)
	patientReceipt, err = testData.DataAPI.GetPatientReceipt(patientVisit.PatientID.Int64(), patientVisit.PatientVisitID.Int64(), patientVisit.SKUType, true)
	test.OK(t, err)
	test.Equals(t, common.PRCharged, patientReceipt.Status)
	test.Equals(t, int64(0), patientReceipt.CreditCardID)
	test.Equals(t, "charge_test1234", patientReceipt.StripeChargeID)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataAPI.GetPatientVisitFromID(patientVisit.PatientVisitID.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}
