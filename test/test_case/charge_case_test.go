package test_case

import (
	"errors"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func TestSucessfulCaseCharge(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, card := setupChargeTest(testData, t)

	// now lets go ahead and start the work to consume the message
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	patient_visit.StartWorker(testData.DataApi, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	time.Sleep(1 * time.Second)

	// at this point there should be a patient receipt, with a stripe charge and a credit card set, the status should be email sent
	patientReceipt, err := testData.DataApi.GetPatientReceipt(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), apiservice.AcneVisit, true)
	test.OK(t, err)
	test.Equals(t, true, patientReceipt != nil)
	test.Equals(t, true, patientReceipt.CreditCardID == card.Id.Int64())
	test.Equals(t, "charge_test", patientReceipt.StripeChargeID)
	test.Equals(t, common.PREmailSent, patientReceipt.Status)
	test.Equals(t, 1, len(patientReceipt.CostBreakdown.LineItems))

	// patient visit should indicate that the message was routed
	patientVisit, err = testData.DataApi.GetPatientVisitFromId(patientVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)

	// there should be a pending item in the unclaimed queue
	dr := test_integration.SignupRandomTestDoctorInState("CA", t, testData)
	eligibleItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(eligibleItems))
}

func TestSuccessfulCharge_AlreadyExists(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, _ := setupChargeTest(testData, t)

	// lets create a receipt and have it exist in a state where its already in the end state
	patientReceipt := &common.PatientReceipt{
		ReferenceNumber: "12345",
		ItemType:        apiservice.AcneVisit,
		ItemID:          patientVisit.PatientVisitId.Int64(),
		PatientID:       patientVisit.PatientId.Int64(),
		Status:          common.PREmailSent,
		CostBreakdown:   &common.CostBreakdown{},
	}
	err := testData.DataApi.CreatePatientReceipt(patientReceipt)
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
	patient_visit.StartWorker(testData.DataApi, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	time.Sleep(1 * time.Second)

	// lets make sure no charge was made and that just one patient receipt exists
	test.Equals(t, false, wasChargeMade)
	var count int
	err = testData.DB.QueryRow(`select count(*) from patient_receipt where patient_id = ?`, patientVisit.PatientId.Int64()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)
	patientReceipt, err = testData.DataApi.GetPatientReceipt(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), apiservice.AcneVisit, true)
	test.OK(t, err)
	test.Equals(t, common.PREmailSent, patientReceipt.Status)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataApi.GetPatientVisitFromId(patientVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}
func TestFailedCharge_StripeFailure(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, card := setupChargeTest(testData, t)

	// lets fail the charge the first time to ensure that message doesn't get routed
	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return nil, errors.New("charge error")
	}

	// set an exceptionally high time period (1 day) so that the worker only runs once
	patient_visit.StartWorker(testData.DataApi, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	time.Sleep(1 * time.Second)

	// at this point the patient receipt should indicate that a charge is still pending
	patientReceipt, err := testData.DataApi.GetPatientReceipt(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), apiservice.AcneVisit, false)
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
	patient_visit.StartWorker(testData.DataApi, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	time.Sleep(time.Second)

	// at this point the charge should go through and there should be just 1 patient receipt existing for the patient
	var count int
	err = testData.DB.QueryRow(`select count(*) from patient_receipt where patient_id = ?`, patientVisit.PatientId.Int64()).Scan(&count)
	test.OK(t, err)
	test.Equals(t, 1, count)
	patientReceipt, err = testData.DataApi.GetPatientReceipt(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), apiservice.AcneVisit, true)
	test.OK(t, err)
	test.Equals(t, common.PREmailSent, patientReceipt.Status)
	test.Equals(t, card.Id.Int64(), patientReceipt.CreditCardID)
	test.Equals(t, "charge_test", patientReceipt.StripeChargeID)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataApi.GetPatientVisitFromId(patientVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}

func TestFailedCharge_ChargeExists(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	patientVisit, stubSQSQueue, _ := setupChargeTest(testData, t)

	// lets create a receipt and have it already exist to simulate a situation where a charge was started but failed for some reason
	patientReceipt := &common.PatientReceipt{
		ReferenceNumber: "12345",
		ItemType:        apiservice.AcneVisit,
		ItemID:          patientVisit.PatientVisitId.Int64(),
		PatientID:       patientVisit.PatientId.Int64(),
		Status:          common.PRChargePending,
		CostBreakdown:   &common.CostBreakdown{},
	}
	err := testData.DataApi.CreatePatientReceipt(patientReceipt)
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
	patient_visit.StartWorker(testData.DataApi, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 24*60*60, "")
	time.Sleep(time.Second)

	test.Equals(t, false, wasCustomerCharged)
	patientReceipt, err = testData.DataApi.GetPatientReceipt(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), apiservice.AcneVisit, true)
	test.OK(t, err)
	test.Equals(t, common.PREmailSent, patientReceipt.Status)
	test.Equals(t, int64(0), patientReceipt.CreditCardID)
	test.Equals(t, "charge_test1234", patientReceipt.StripeChargeID)

	// patient visit should indicate that it was charged
	patientVisit, err = testData.DataApi.GetPatientVisitFromId(patientVisit.PatientVisitId.Int64())
	test.OK(t, err)
	test.Equals(t, common.PVStatusRouted, patientVisit.Status)
}

func setupChargeTest(testData *test_integration.TestData, t *testing.T) (*common.PatientVisit, *common.SQSQueue, *common.Card) {
	// lets introduce a cost for an acne visit
	res, err := testData.DB.Exec(`insert into item_cost (item_type, status) values (?,?)`, apiservice.AcneVisit, api.STATUS_ACTIVE)
	test.OK(t, err)
	itemCostId, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Visit','40.00',?)`, itemCostId)
	test.OK(t, err)

	stubSQSQueue := &common.SQSQueue{
		QueueUrl:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}

	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)

	// now lets go ahead and submit a visit
	pv := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(pv.PatientVisitId)
	test.OK(t, err)

	// lets go ahead and add a default card for the patient
	card := &common.Card{
		ThirdPartyId: "thirdparty",
		Fingerprint:  "fingerprint",
		Token:        "token",
		Type:         "Visa",
		BillingAddress: &common.Address{
			AddressLine1: "addressLine1",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94115",
		},
	}
	testData.DataApi.AddCardAndMakeDefaultForPatient(patientVisit.PatientId.Int64(), card)
	return patientVisit, stubSQSQueue, card
}
