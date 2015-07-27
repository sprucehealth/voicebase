package test_promotions

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotionReferralRouteCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	setupPromotionReferralRouteTest(t, testData)

	code := "TestPromotionReferralRoute"
	promoCodeID, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code,
		Group: "new_user",
		Data: promotions.NewPercentOffVisitPromotion(
			100,
			"new_user",
			"displayMsg",
			"shortMsg",
			"successMsg",
			"imageURL",
			1,
			1,
			true),
	})
	test.OK(t, err)

	promoCode, err := testData.DataAPI.LookupPromoCode(code)
	test.OK(t, err)
	test.Equals(t, promoCodeID, promoCode.ID)

	gender := common.PRRGender("M")
	al := 1
	au := 1000
	state := "FL"
	pharmacy := "CVS"
	id, err := testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCode.ID,
		Priority:        100,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        &al,
		AgeUpper:        &au,
		State:           &state,
		Pharmacy:        &pharmacy,
	})
	test.OK(t, err)

	routes, err := testData.DataAPI.PromotionReferralRoutes([]string{"ACTIVE"})
	test.OK(t, err)
	test.Equals(t, 1, len(routes))
	test.Equals(t, id, routes[0].ID)
	test.Equals(t, common.PRRLifecycle("ACTIVE"), routes[0].Lifecycle)
	test.Equals(t, 100, routes[0].Priority)
	test.Equals(t, gender, *routes[0].Gender)
	test.Equals(t, al, *routes[0].AgeLower)
	test.Equals(t, au, *routes[0].AgeUpper)
	test.Equals(t, state, *routes[0].State)
	test.Equals(t, pharmacy, *routes[0].Pharmacy)
}

func TestPromotionReferralRouteQueryParamCreation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	setupPromotionReferralRouteTest(t, testData)

	pvr := test_integration.CreateRandomPatientVisitInState("FL", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)
	params, err := testData.DataAPI.RouteQueryParamsForAccount(patient.AccountID.Int64())
	test.OK(t, err)
	test.Equals(t, "FL", *params.State)
	test.Assert(t, params.Age != nil, "Expected a non null age calculation")
	test.Equals(t, "Test Pharmacy", *params.Pharmacy)
	test.Assert(t, params.Gender != nil, "Expected a non null gender")
}

func TestPromotionReferralRouteQueryParamsToTemplate(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	setupPromotionReferralRouteTest(t, testData)

	promo := promotions.NewPercentOffVisitPromotion(
		100,
		"new_user",
		"displayMsg",
		"shortMsg",
		"successMsg",
		"imageURL",
		1,
		1,
		true)
	rp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, nil, "", 0, 0)
	test.OK(t, err)
	code1 := "TestPromotionReferralRoute1"
	promoCodeID1, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code1,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	code2 := "TestPromotionReferralRoute2"
	promoCodeID2, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code2,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	tid1, err := testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Active"),
		PromotionCodeID: &promoCodeID1,
		Data:            rp,
	})
	test.OK(t, err)

	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Active"),
		PromotionCodeID: &promoCodeID2,
		Data:            rp,
	})
	test.OK(t, err)

	gender := common.PRRGender("M")
	al := 1
	au := 1000
	state := "FL"
	rid, err := testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCodeID1,
		Priority:        100,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        &al,
		AgeUpper:        &au,
		State:           &state,
		Pharmacy:        nil,
	})
	test.OK(t, err)
	_, err = testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCodeID2,
		Priority:        101,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        nil,
		AgeUpper:        nil,
		State:           nil,
		Pharmacy:        nil,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("FL", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)
	params, err := testData.DataAPI.RouteQueryParamsForAccount(patient.AccountID.Int64())
	test.OK(t, err)
	routeID, template, err := testData.DataAPI.ReferralProgramTemplateRouteQuery(params)
	test.OK(t, err)
	test.Equals(t, tid1, template.ID)
	test.Equals(t, rid, *routeID)
}

func TestPromotionReferralRouteQueryParamsToInactiveTemplate(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	setupPromotionReferralRouteTest(t, testData)

	promo := promotions.NewPercentOffVisitPromotion(
		100,
		"new_user",
		"displayMsg",
		"shortMsg",
		"successMsg",
		"imageURL",
		1,
		1,
		true)
	rp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, nil, "", 0, 0)
	test.OK(t, err)
	code1 := "TestPromotionReferralRoute1"
	promoCodeID1, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code1,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	code2 := "TestPromotionReferralRoute2"
	promoCodeID2, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code2,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Inactive"),
		PromotionCodeID: &promoCodeID1,
		Data:            rp,
	})
	test.OK(t, err)

	defaultTemplateID, err := testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Default"),
		PromotionCodeID: &promoCodeID2,
		Data:            rp,
	})
	test.OK(t, err)

	gender := common.PRRGender("M")
	al := 1
	au := 1000
	state := "FL"
	_, err = testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCodeID1,
		Priority:        100,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        &al,
		AgeUpper:        &au,
		State:           &state,
		Pharmacy:        nil,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("FL", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)
	params, err := testData.DataAPI.RouteQueryParamsForAccount(patient.AccountID.Int64())
	test.OK(t, err)
	routeID, template, err := testData.DataAPI.ReferralProgramTemplateRouteQuery(params)
	test.OK(t, err)
	test.Equals(t, defaultTemplateID, template.ID)
	test.Assert(t, routeID == nil, "Expected nil route id")
}

func TestPromotionReferralRouteDeprecationDisplayFullLoop(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	setupPromotionReferralRouteTest(t, testData)

	promo := promotions.NewPercentOffVisitPromotion(
		100,
		"new_user",
		"displayMsg",
		"shortMsg",
		"successMsg",
		"imageURL",
		1,
		1,
		true)
	rp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, &promotions.ShareTextParams{}, "", 0, 0)
	test.OK(t, err)
	code1 := "TestPromotionReferralRoute1"
	promoCodeID1, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code1,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Active"),
		PromotionCodeID: &promoCodeID1,
		Data:            rp,
	})
	test.OK(t, err)

	gender := common.PRRGender("M")
	al := 1
	au := 1000
	state := "FL"
	rid, err := testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCodeID1,
		Priority:        100,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        &al,
		AgeUpper:        &au,
		State:           &state,
		Pharmacy:        nil,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("FL", t, testData)
	patient, err := testData.DataAPI.GetPatientFromPatientVisitID(pvr.PatientVisitID)
	test.OK(t, err)

	promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.local", patient.AccountID.Int64())

	arp, err := testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, *arp.PromotionReferralRouteID, rid)

	af, err := testData.DataAPI.UpdatePromotionReferralRoute(&common.PromotionReferralRouteUpdate{
		ID:        rid,
		Lifecycle: common.PRRLifecycle("DEPRECATED"),
	})
	test.OK(t, err)
	test.Equals(t, int64(1), af)

	_, err = testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.Assert(t, api.IsErrNotFound(err), "Expected no active RP to be found after deprecating the route")

	af, err = testData.DataAPI.UpdatePromotionReferralRoute(&common.PromotionReferralRouteUpdate{
		ID:        rid,
		Lifecycle: common.PRRLifecycle("ACTIVE"),
	})
	test.OK(t, err)
	test.Equals(t, int64(1), af)

	rid2, err := testData.DataAPI.InsertPromotionReferralRoute(&common.PromotionReferralRoute{
		PromotionCodeID: promoCodeID1,
		Priority:        101,
		Lifecycle:       common.PRRLifecycle("ACTIVE"),
		Gender:          &gender,
		AgeLower:        &al,
		AgeUpper:        &au,
		State:           &state,
		Pharmacy:        nil,
	})

	promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.local", patient.AccountID.Int64())

	arp, err = testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, rid2, *arp.PromotionReferralRouteID)
}

func setupPromotionReferralRouteTest(t *testing.T, testData *test_integration.TestData) {
	_, err := testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "new_user",
		MaxAllowedPromos: 1,
	})
	test.OK(t, err)
}
