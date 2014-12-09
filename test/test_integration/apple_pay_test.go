package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
)

func TestApplePay(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	// setup the test to have a cost for the acne_visit SKU
	// so that the card is actually charged
	SetupActiveCostForAcne(testData, t)
	stubSQSQueue := &common.SQSQueue{
		QueueURL:     "visit_url",
		QueueService: &sqs.StubSQS{},
	}
	testData.Config.VisitQueue = stubSQSQueue
	testData.StartAPIServer(t)

	customerToAdd := &stripe.Customer{
		ID: "test_customer_id",
		CardList: &stripe.CardList{
			Cards: []*stripe.Card{
				{
					ID:          "third_party_id0",
					Fingerprint: "test_fingerprint0",
				},
			},
		},
	}
	stubPaymentsService := testData.Config.PaymentAPI.(*StripeStub)
	stubPaymentsService.CustomerToReturn = customerToAdd
	stubPaymentsService.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}

	// setup the patient to be in a state where the visit can be submitted
	signedupPatientResponse := SignupRandomTestPatient(t, testData)
	AddTestPharmacyForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)
	AddTestAddressForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)

	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientID.Int64(), testData, t)

	req := &patient.PatientVisitRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitID,
		Card: &common.Card{
			Token: "1235 " + strconv.FormatInt(time.Now().UnixNano(), 10),
			Type:  "ApplePay",
			BillingAddress: &common.Address{
				AddressLine1: "1234 Main Street " + strconv.FormatInt(time.Now().UnixNano(), 10),
				AddressLine2: "Apt 12345",
				City:         "San Francisco",
				State:        "California",
				ZipCode:      "12345",
			},
		},
		ApplePay: true,
	}

	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(req); err != nil {
		t.Fatal(err)
	}

	// submit the visit with a card specified
	resp, err := testData.AuthPut(testData.APIServer.URL+apipaths.PatientVisitURLPath,
		"application/json", body, signedupPatientResponse.Patient.AccountID.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// make sure that the card is the default card on file, and that its got apple pay set to 1
	cards, err := testData.DataAPI.GetCardsForPatient(signedupPatientResponse.Patient.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, 1, len(cards))
	test.Equals(t, true, cards[0].ApplePay)

	// start the worker to charge the card that the patient submitted the visit with
	w := cost.StartWorker(testData.DataAPI, testData.Config.AnalyticsLogger,
		testData.Config.Dispatcher, stubPaymentsService, nil, stubSQSQueue, metrics.NewRegistry(), 0, "")
	defer w.Stop()

	ok := false
	for try := 0; try < 10; try++ {
		time.Sleep(time.Millisecond * 100)
		visit, err := testData.DataAPI.GetPatientVisitFromID(patientVisitResponse.PatientVisitID)
		if err != nil {
			t.Fatal(err)
		}
		if visit.Status == "ROUTED" {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatal("Visit never routed")
	}
}
