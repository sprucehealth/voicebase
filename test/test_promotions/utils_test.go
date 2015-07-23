package test_promotions

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

var globalFirstVisitFreeDisabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "A value that represents if the first visit should be free for all patients.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

func createPromotion(promotion promotions.Promotion, testData *test_integration.TestData, t *testing.T) string {
	promoCode, err := promotions.GeneratePromoCode(testData.DataAPI)
	test.OK(t, err)
	test.Equals(t, true, promoCode != "")

	_, err = testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  promoCode,
		Data:  promotion,
		Group: promotion.Group(),
	})
	test.OK(t, err)
	return promoCode
}

func setupPromotionsTest(testData *test_integration.TestData, t *testing.T) {
	// lets introduce a cost for an acne visit
	var skuID int64
	err := testData.DB.QueryRow(`select id from sku where type = 'acne_visit'`).Scan(&skuID)
	test.OK(t, err)

	res, err := testData.DB.Exec(`insert into item_cost (sku_id, status) values (?,?)`, skuID, api.StatusActive)
	test.OK(t, err)
	itemCostID, err := res.LastInsertId()
	test.OK(t, err)
	_, err = testData.DB.Exec(`insert into line_item (currency, description, amount, item_cost_id) values ('USD','Acne Visit',4000,?)`, itemCostID)
	test.OK(t, err)

	// lets add a prefix to generate random codes with
	err = testData.DataAPI.CreatePromoCodePrefix("SpruceUp")
	test.OK(t, err)

	// lets create a promo group
	_, err = testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "new_user",
		MaxAllowedPromos: 1,
	})
	test.OK(t, err)
}

func startAndSubmitVisit(patientID int64, patientAccountID int64,
	stubSQSQueue *common.SQSQueue, testData *test_integration.TestData, t *testing.T) int64 {
	pv := test_integration.CreatePatientVisitForPatient(patientID, testData, t)
	answerIntake := test_integration.PrepareAnswersForQuestionsInPatientVisit(pv.PatientVisitID, pv.ClientLayout.InfoIntakeLayout, t)
	test_integration.SubmitAnswersIntakeForPatient(patientID, patientAccountID, answerIntake, testData, t)

	stubStripe := testData.Config.PaymentAPI.(*test_integration.StripeStub)
	stubStripe.CreateChargeFunc = func(req *stripe.CreateChargeRequest) (*stripe.Charge, error) {
		return &stripe.Charge{
			ID: "charge_test",
		}, nil
	}
	test_integration.SubmitPatientVisitForPatient(patientID, pv.PatientVisitID, testData, t)

	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{globalFirstVisitFreeDisabled})
	test.OK(t, err)
	w := cost.NewWorker(testData.DataAPI, testData.Config.AnalyticsLogger, testData.Config.Dispatcher, stubStripe, nil, stubSQSQueue, metrics.NewRegistry(), 0, "", cfgStore)
	w.Do()
	return pv.PatientVisitID
}

func getPatientReceipt(patientID, patientVisitID int64, testData *test_integration.TestData, t *testing.T) *common.PatientReceipt {
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(patientVisitID)
	test.OK(t, err)
	patientReciept, err := testData.DataAPI.GetPatientReceipt(patientID, patientVisitID, patientVisit.SKUType, true)
	test.OK(t, err)
	patientReciept.CostBreakdown.CalculateTotal()
	return patientReciept
}

// Note: Reason for this helper method versus using the shared utility methods from test_integration to create patients
// is because for some of the promotions we need to assume that the visit has been created at the time of signup (which is
// what is happening in most cases). This is because the route doctor promotion assumes the existence of a case to assign the doctor
// from the promotion to the case care team.
func signupPatientWithVisit(email string, testData *test_integration.TestData, t *testing.T) *patient.PatientSignedupResponse {
	// lets signup a patient with state code provided
	params := url.Values{}
	params.Set("first_name", "test")
	params.Set("last_name", "test1")
	params.Set("email", email)
	params.Set("password", "12345")
	params.Set("state_code", "CA")
	params.Set("zip_code", "94115")
	params.Set("dob", "1987-11-08")
	params.Set("gender", "female")
	params.Set("phone", "2068773590")
	params.Set("create_visit", "true")

	req, err := http.NewRequest("POST", testData.APIServer.URL+apipaths.PatientSignupURLPath, strings.NewReader(params.Encode()))
	test.OK(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("S-Version", "Patient;Dev;0.9.5")
	req.Header.Set("S-OS", "iOS;")
	resp, err := http.DefaultClient.Do(req)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	var respData patient.PatientSignedupResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	test.OK(t, err)
	test.Equals(t, true, respData.PatientVisitData != nil)
	return &respData
}
