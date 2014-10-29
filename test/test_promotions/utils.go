package test_promotions

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/stripe"
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

func getPatientReceipt(patientID, patientVisitID int64, testData *test_integration.TestData, t *testing.T) *common.PatientReceipt {
	patientVisit, err := testData.DataApi.GetPatientVisitFromId(patientVisitID)
	test.OK(t, err)
	patientReciept, err := testData.DataApi.GetPatientReceipt(patientID, patientVisitID, patientVisit.SKU, true)
	test.OK(t, err)
	patientReciept.CostBreakdown.CalculateTotal()
	return patientReciept
}
