package test_promotions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPromoCodeConfirmation_Promotion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	promoCode := CreateRandomPromotion(t, testData, nil, `{
    "display_msg": "display_msg",
    "image_url": "image_url",
    "short_msg": "short_msg",
    "success_msg": "success_msg",
    "group": "new_user",
    "for_new_user": false,
    "value": 25
  }`, `promo_money_off`)

	res, err := patientClient.PromotionConfirmation(&promotions.PromotionConfirmationGETRequest{
		Code: promoCode,
	})
	test.OK(t, err)
	test.Equals(t, "Congratulations!", res.Title)
	test.Equals(t, "spruce:///image/icon_case_large", res.ImageURL)
	test.Equals(t, "success_msg", res.BodyText)
	test.Equals(t, "Let's Go", res.ButtonTitle)
}

func TestPromoCodeConfirmation_PatientReferral(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	CreateReferralProgram(t, testData)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	patientClient := test_integration.PatientClient(testData, t, patientVisit.PatientID.Int64())
	patients, err := testData.DataAPI.GetPatientsForIDs([]int64{patientVisit.PatientID.Int64()})
	test.OK(t, err)
	test.Assert(t, len(patients) == 1, "Expected only 1 patient to be returned but got %d", len(patients))

	referralProgramTemplate, err := testData.DataAPI.ActiveReferralProgramTemplate(api.RolePatient, common.PromotionTypes)
	test.OK(t, err)

	_, err = promotions.CreateReferralProgramFromTemplate(referralProgramTemplate, patients[0].AccountID.Int64(), testData.DataAPI)
	test.OK(t, err)

	rp, err := testData.DataAPI.ActiveReferralProgramForAccount(patients[0].AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Assert(t, rp != nil, "Expected an active referral program")

	res, err := patientClient.PromotionConfirmation(&promotions.PromotionConfirmationGETRequest{
		Code: rp.Code,
	})
	test.OK(t, err)
	test.Equals(t, fmt.Sprintf("Your friend %s has given you a free visit.", patients[0].FirstName), res.Title)
	test.Equals(t, "spruce:///image/icon_case_large", res.ImageURL)
	test.Equals(t, "success_msg", res.BodyText)
	test.Equals(t, "Let's Go", res.ButtonTitle)
}

func TestPromoCodeConfirmation_DoctorReferral(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr, email, password := test_integration.SignupRandomTestDoctor(t, testData)
	dcli := test_integration.DoctorClient(testData, t, dr.DoctorID)
	_, err := dcli.Auth(email, password)
	test.OK(t, err)
	pcli := test_integration.PatientClient(testData, t, 0)

	doctor, err := testData.DataAPI.GetDoctorFromID(dr.DoctorID)
	test.OK(t, err)

	// at this point there should be a referral program for the doctor
	rp, err := testData.DataAPI.ActiveReferralProgramForAccount(doctor.AccountID.Int64(), common.PromotionTypes)
	test.OK(t, err)
	test.Equals(t, true, rp != nil)

	res, err := pcli.PromotionConfirmation(&promotions.PromotionConfirmationGETRequest{
		Code: rp.Code,
	})
	test.OK(t, err)
	test.Equals(t, "Welcome to Spruce!", res.Title)
	test.Equals(t, "spruce:///image/icon_case_large", res.ImageURL)
	test.Equals(t, fmt.Sprintf("You will be seen by Dr. %s %s.", doctor.FirstName, doctor.LastName), res.BodyText)
	test.Equals(t, "Let's Go", res.ButtonTitle)
}

func CreateReferralProgram(t *testing.T, testData *test_integration.TestData) {
	admin := test_integration.CreateRandomAdmin(t, testData)

	// create referral program template
	requestData := map[string]interface{}{
		"type":        "promo_money_off",
		"title":       "title",
		"description": "description",
		"group":       "new_user",
		"share_text": map[string]interface{}{
			"facebook": "facebook",
			"sms":      "sms",
			"default":  "default",
		},
		"promotion": map[string]interface{}{
			"display_msg":  "display_msg",
			"success_msg":  "success_msg",
			"short_msg":    "short_msg",
			"for_new_user": true,
			"group":        "new_user",
			"value":        50,
		},
	}

	var responseData map[string]interface{}
	resp, err := testData.AuthPostJSON(testData.APIServer.URL+apipaths.ReferralProgramsTemplateURLPath, admin.AccountID.Int64(), requestData, &responseData)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}
