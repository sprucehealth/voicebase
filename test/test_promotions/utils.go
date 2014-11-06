package test_promotions

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/sku"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

func createPromotion(promotion promotions.Promotion, testData *test_integration.TestData, t *testing.T) string {
	promoCode, err := promotions.GeneratePromoCode(testData.DataApi)
	test.OK(t, err)
	test.Equals(t, true, promoCode != "")

	err = testData.DataApi.CreatePromotion(&common.Promotion{
		Code:  promoCode,
		Data:  promotion,
		Group: promotion.Group(),
	})
	test.OK(t, err)
	return promoCode
}

func setupPromotionsTest(testData *test_integration.TestData, t *testing.T) {
	// lets introduce a cost for an acne visit
	var skuId int64
	err := testData.DB.QueryRow(`select id from sku where type = 'acne_visit'`).Scan(&skuId)
	test.OK(t, err)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuId, api.STATUS_ACTIVE)
	test.OK(t, err)
	itemCostId, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Visit',4000,?)`, itemCostId)
	test.OK(t, err)

	// lets add a prefix to generate random codes with
	err = testData.DataApi.CreatePromoCodePrefix("SpruceUp")
	test.OK(t, err)

	// lets create a promo group
	_, err = testData.DataApi.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "new_user",
		MaxAllowedPromos: 1,
	})
	test.OK(t, err)
}

func startAndSubmitVisit(patientID int64, patientAccountID int64,
	stubSQSQueue *common.SQSQueue, testData *test_integration.TestData, t *testing.T) (*cost.Worker, int64) {
	pv := test_integration.CreatePatientVisitForPatient(patientID, testData, t)
	answerIntake := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv, t)
	test_integration.SubmitAnswersIntakeForPatient(patientID, patientAccountID, answerIntake, testData, t)

	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}
	test_integration.SubmitPatientVisitForPatient(patientID, pv.PatientVisitId, testData, t)
	w := cost.StartWorker(testData.DataApi, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 0, "")
	time.Sleep(500 * time.Millisecond)
	return w, pv.PatientVisitId
}

type lineItem struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type costResponse struct {
	Total     *lineItem   `json:"total"`
	LineItems []*lineItem `json:"line_items"`
}

func queryCost(patientAccountID int64, testData *test_integration.TestData, t *testing.T) (string, []*lineItem) {
	res, err := testData.AuthGet(testData.APIServer.URL+router.PatientCostURLPath+"?item_type=acne_visit", patientAccountID)
	test.OK(t, err)
	defer res.Body.Close()
	test.Equals(t, http.StatusOK, res.StatusCode)
	var response costResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	test.OK(t, err)
	return response.Total.Value, response.LineItems
}

func getPatientReceipt(patientID, patientVisitID int64, testData *test_integration.TestData, t *testing.T) *common.PatientReceipt {
	patientReciept, err := testData.DataApi.GetPatientReceipt(patientID, patientVisitID, sku.AcneVisit, true)
	test.OK(t, err)
	patientReciept.CostBreakdown.CalculateTotal()
	return patientReciept
}

func addCreditCardForPatient(patientID int64, testData *test_integration.TestData, t *testing.T) {
	err := testData.DataApi.AddCardForPatient(patientID, &common.Card{
		ThirdPartyID: "thirdparty",
		Fingerprint:  "fingerprint",
		Token:        "token",
		Type:         "Visa",
		BillingAddress: &common.Address{
			AddressLine1: "addressLine1",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94115",
		},
		IsDefault: true,
	})
	test.OK(t, err)
}
