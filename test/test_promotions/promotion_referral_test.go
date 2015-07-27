package test_promotions

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromotionPercentReferralUpdates(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionReferralTest(t, testData)

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

	code := "TestPromotionPercent"
	promoCodeID, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	grp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, &promotions.ShareTextParams{}, "", 0, 0)
	test.OK(t, err)
	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Default"),
		PromotionCodeID: &promoCodeID,
		Data:            grp,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patient, err := testData.DataAPI.Patient(patientVisit.PatientID.Int64(), false)
	test.OK(t, err)

	_, err = promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.local", patient.AccountID.Int64())
	test.OK(t, err)

	rp, err := testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	referralProgram := rp.Data.(promotions.ReferralProgram)

	test.OK(t, referralProgram.ReferredAccountAssociatedCode(patient.AccountID.Int64(), rp.CodeID, testData.DataAPI))

	rp, err = testData.DataAPI.ReferralProgram(rp.CodeID, common.PromotionTypes)
	test.OK(t, err)
	referralProgram = rp.Data.(promotions.ReferralProgram)
	test.Assert(t, referralProgram.PromotionForReferredAccount(rp.Code) != nil, "Expected promo to be non nil after update")

	test.OK(t, referralProgram.ReferredAccountSubmittedVisit(patient.AccountID.Int64(), rp.CodeID, testData.DataAPI))

	rp, err = testData.DataAPI.ReferralProgram(rp.CodeID, common.PromotionTypes)
	test.OK(t, err)
	referralProgram = rp.Data.(promotions.ReferralProgram)
	test.Assert(t, referralProgram.PromotionForReferredAccount(rp.Code) != nil, "Expected promo to be non nil after update")
}

func TestPromotionMoneyReferralUpdates(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionReferralTest(t, testData)

	promo := promotions.NewMoneyOffVisitPromotion(
		100,
		"new_user",
		"displayMsg",
		"shortMsg",
		"successMsg",
		"imageURL",
		1,
		1,
		true)

	code := "TestPromotionMoney"
	promoCodeID, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	grp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, &promotions.ShareTextParams{}, "referralImage", 120, 130)
	test.OK(t, err)
	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Default"),
		PromotionCodeID: &promoCodeID,
		Data:            grp,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patient, err := testData.DataAPI.Patient(patientVisit.PatientID.Int64(), false)
	test.OK(t, err)

	display, err := promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.local", patient.AccountID.Int64())
	test.OK(t, err)
	test.Equals(t, display.ImageURL, "referralImage")
	test.Equals(t, display.ImageWidth, 120)
	test.Equals(t, display.ImageHeight, 130)

	rp, err := testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	referralProgram := rp.Data.(promotions.ReferralProgram)

	test.OK(t, referralProgram.ReferredAccountAssociatedCode(patient.AccountID.Int64(), rp.CodeID, testData.DataAPI))

	rp, err = testData.DataAPI.ReferralProgram(rp.CodeID, common.PromotionTypes)
	test.OK(t, err)
	referralProgram = rp.Data.(promotions.ReferralProgram)
	test.Assert(t, referralProgram.PromotionForReferredAccount(rp.Code) != nil, "Expected promo to be non nil after update")

	test.OK(t, referralProgram.ReferredAccountSubmittedVisit(patient.AccountID.Int64(), rp.CodeID, testData.DataAPI))

	rp, err = testData.DataAPI.ReferralProgram(rp.CodeID, common.PromotionTypes)
	test.OK(t, err)
	referralProgram = rp.Data.(promotions.ReferralProgram)
	test.Assert(t, referralProgram.PromotionForReferredAccount(rp.Code) != nil, "Expected promo to be non nil after update")
}

func TestPromotionApplyOwnReferralCode(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	setupPromotionReferralTest(t, testData)

	promo := promotions.NewMoneyOffVisitPromotion(
		100,
		"new_user",
		"displayMsg",
		"shortMsg",
		"successMsg",
		"imageURL",
		1,
		1,
		true)

	code := "TestPromotionMoney"
	promoCodeID, err := testData.DataAPI.CreatePromotion(&common.Promotion{
		Code:  code,
		Group: "new_user",
		Data:  promo,
	})
	test.OK(t, err)

	grp, err := promotions.NewGiveReferralProgram("title", "description", "group", nil, promo, &promotions.ShareTextParams{}, "", 0, 0)
	test.OK(t, err)
	_, err = testData.DataAPI.CreateReferralProgramTemplate(&common.ReferralProgramTemplate{
		Role:            api.RolePatient,
		Status:          common.ReferralProgramStatus("Default"),
		PromotionCodeID: &promoCodeID,
		Data:            grp,
	})
	test.OK(t, err)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patient, err := testData.DataAPI.Patient(patientVisit.PatientID.Int64(), false)
	test.OK(t, err)

	promotions.CreateReferralDisplayInfo(testData.DataAPI, "www.spruce.local", patient.AccountID.Int64())

	rp, err := testData.DataAPI.ActiveReferralProgramForAccount(patient.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	_, err = patientClient.ApplyPromoCode(&promotions.PatientPromotionPOSTRequest{
		PromoCode: rp.Code,
	})
	sperr, ok := err.(*apiservice.SpruceError)
	test.Assert(t, ok, "Could not convert to spruce error")
	test.Equals(t, http.StatusNotFound, sperr.HTTPStatusCode)

	var status string
	test.Equals(t, testData.DB.QueryRow(`SELECT status FROM account_promotion WHERE account_id = ?`, patient.AccountID.Int64()).Scan(&status), sql.ErrNoRows)
}

func setupPromotionReferralTest(t *testing.T, testData *test_integration.TestData) {
	_, err := testData.DataAPI.CreatePromotionGroup(&common.PromotionGroup{
		Name:             "new_user",
		MaxAllowedPromos: 1,
	})
	test.OK(t, err)
}
