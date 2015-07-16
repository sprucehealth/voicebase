package test_promotions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientPromoCode_ApplyPromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	res, err := patientClient.ActivePromotions()
	test.OK(t, err)
	test.Equals(t, 0, len(res.ActivePromotions))
	test.Equals(t, 0, len(res.ExpiredPromotions))

	promoCode := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)

	res, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode,
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.ActivePromotions))
	test.Equals(t, 0, len(res.ExpiredPromotions))
}

func TestPatientPromoCode_ApplyExpiredPromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	past := time.Unix(time.Now().Unix()-1000, 0)
	promoCode := CreateRandomPromotion(t, testData, &past, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)

	_, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode,
	})
	test.Assert(t, err != nil, "Expected an error for applying an expired promo code")
	serror, ok := err.(*apiservice.SpruceError)
	test.Assert(t, ok, "Unable to case as SpruceError")
	test.Equals(t, http.StatusNotFound, serror.HTTPStatusCode)
}

func TestPatientPromoCode_ApplyZerovaluePromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	res, err := patientClient.ActivePromotions()
	test.OK(t, err)
	test.Equals(t, 0, len(res.ActivePromotions))
	test.Equals(t, 0, len(res.ExpiredPromotions))

	promoCode := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 0
  }`, `promo_money_off`)

	_, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode,
	})
	sperr, ok := err.(*apiservice.SpruceError)
	test.Assert(t, ok, "Could not convert to spruce error")
	test.Equals(t, http.StatusNotFound, sperr.HTTPStatusCode)

	patient, err := testData.DataAPI.Patient(patientVisit.PatientID.Int64(), true)
	test.OK(t, err)

	var status string
	test.OK(t, testData.DB.QueryRow(`SELECT status FROM account_promotion WHERE account_id = ?`, patient.AccountID.Int64()).Scan(&status))
	test.Equals(t, common.PSPending.String(), status)
}

func TestPatientPromoCode_ApplyBadPromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	_, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: "DoesNotExist",
	})
	test.Assert(t, err != nil, "Expected an error for applying an expired promo code")
	serror, ok := err.(*apiservice.SpruceError)
	test.Assert(t, ok, "Unable to case as SpruceError")
	test.Equals(t, http.StatusNotFound, serror.HTTPStatusCode)
}

func TestPatientPromoCode_ApplyMultiplePromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	res, err := patientClient.ActivePromotions()
	test.OK(t, err)
	test.Equals(t, 0, len(res.ActivePromotions))
	test.Equals(t, 0, len(res.ExpiredPromotions))

	promoCode := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)

	res, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode,
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.ActivePromotions))
	test.Equals(t, promoCode, res.ActivePromotions[0].Code)
	test.Equals(t, 0, len(res.ExpiredPromotions))

	promoCode2 := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)
	test.Assert(t, promoCode != promoCode2, "Expected different promo codes to be generated")

	res, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode2,
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.ActivePromotions))
	test.Equals(t, promoCode2, res.ActivePromotions[0].Code)
	test.Equals(t, 0, len(res.ExpiredPromotions))
}

func TestPatientPromoCode_ExistingExpiredPromotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	res, err := patientClient.ActivePromotions()
	test.OK(t, err)
	test.Equals(t, 0, len(res.ActivePromotions))
	test.Equals(t, 0, len(res.ExpiredPromotions))

	promoCode := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)

	res, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: promoCode,
	})
	test.OK(t, err)
	test.Equals(t, 1, len(res.ActivePromotions))
	test.Equals(t, promoCode, res.ActivePromotions[0].Code)
	test.Equals(t, 0, len(res.ExpiredPromotions))

	result, err := testData.DB.Exec("UPDATE account_promotion SET expires = ? WHERE promotion_code_id IN (SELECT id FROM promotion_code WHERE code = ?)", time.Unix(time.Now().Unix()-1000, 0), promoCode)
	test.OK(t, err)

	aff, err := result.RowsAffected()
	test.OK(t, err)
	test.Equals(t, int64(1), aff)

	res, err = patientClient.ActivePromotions()
	test.OK(t, err)
	test.Equals(t, 0, len(res.ActivePromotions))
	test.Equals(t, 1, len(res.ExpiredPromotions))
	test.Equals(t, promoCode, res.ExpiredPromotions[0].Code)
}

func CreateRandomPromotion(t *testing.T, testData *test_integration.TestData, expiration *time.Time, promotion, promoType string) string {
	promoCode, err := promotions.GeneratePromoCode(testData.DataAPI)
	test.OK(t, err)

	promotionDataType, ok := common.PromotionTypes[promoType]
	if !ok {
		test.OK(t, fmt.Errorf("Unknown type "+promoType))
	}

	promotionData := reflect.New(promotionDataType).Interface().(promotions.Promotion)
	err = json.Unmarshal([]byte(promotion), &promotionData)
	test.OK(t, err)

	_, err = testData.DataAPI.PromotionGroup(promotionData.Group())
	if api.IsErrNotFound(err) {
		_, err = testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
			Name:             promotionData.Group(),
			MaxAllowedPromos: 99,
		})
		test.OK(t, err)
	}
	test.OK(t, err)

	promo := &common.Promotion{
		Code:    promoCode,
		Data:    promotionData,
		Group:   promotionData.Group(),
		Expires: expiration,
	}

	_, err = testData.DataAPI.CreatePromotion(promo)
	test.OK(t, err)
	return promoCode
}
