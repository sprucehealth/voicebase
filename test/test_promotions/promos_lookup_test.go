package test_promotions

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotion_Lookup(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	// create group
	_, err := testData.DataApi.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "new_user",
		MaxAllowedPromos: 1,
	})
	test.OK(t, err)

	// lets create a promotion
	displayMsg := "5% off visit for new Spruce Users"
	promotion := promotions.NewPercentOffVisitPromotion(5,
		"new_user",
		displayMsg,
		displayMsg,
		"Successfully claimed 5% coupon code",
		true)
	promoCode := createPromotion(promotion, testData, t)

	// now lets look it up
	displayInfo, err := promotions.LookupPromoCode(promoCode, testData.DataApi, testData.Config.AnalyticsLogger)
	test.OK(t, err)
	test.Equals(t, true, displayInfo != nil)
	test.Equals(t, displayMsg, displayInfo.Title)

	// lets look up non-existent group
	displayInfo, err = promotions.LookupPromoCode("123", testData.DataApi, testData.Config.AnalyticsLogger)
	test.Equals(t, promotions.InvalidCode, err)
	test.Equals(t, true, displayInfo == nil)

	// lets an expired promotion
	promoCode, err = promotions.GeneratePromoCode(testData.DataApi)
	test.OK(t, err)

	inThePast := time.Now().Add(-5 * time.Hour)
	err = testData.DataApi.CreatePromotion(&common.Promotion{
		Code:    promoCode,
		Data:    promotion,
		Group:   promotion.Group(),
		Expires: &inThePast,
	})
	test.OK(t, err)
	displayInfo, err = promotions.LookupPromoCode(promoCode, testData.DataApi, testData.Config.AnalyticsLogger)
	test.Equals(t, promotions.PromotionExpired, err)
	test.Equals(t, true, displayInfo == nil)
}
