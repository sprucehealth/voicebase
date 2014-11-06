package test_integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
)

func TestApplePay(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	customerToAdd := &stripe.Customer{
		Id: "test_customer_id",
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

	signedupPatientResponse := SignupRandomTestPatient(t, testData)
	patientVisitResponse := CreatePatientVisitForPatient(signedupPatientResponse.Patient.PatientId.Int64(), testData, t)

	req := &patient.PatientVisitRequestData{
		PatientVisitID: patientVisitResponse.PatientVisitId,
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

	resp, err := testData.AuthPut(testData.APIServer.URL+router.PatientVisitURLPath,
		"application/json", body, signedupPatientResponse.Patient.AccountId.Int64())
	test.OK(t, err)
	resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	ok := false
	for try := 0; try < 10; try++ {
		time.Sleep(time.Millisecond * 100)
		visit, err := testData.DataApi.GetPatientVisitFromId(patientVisitResponse.PatientVisitId)
		if err != nil {
			t.Fatal(err)
		}
		if visit.Status == "ROUTED" {
			ok = true
			break
		} else if visit.Status != "OPEN" {
			t.Fatal("Unexpected visit status: " + visit.Status)
		}
	}
	if !ok {
		t.Fatal("Visit never routed")
	}
}
