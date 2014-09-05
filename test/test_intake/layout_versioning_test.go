package test_intake

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestLayoutVersioning_MajorUpgrade(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// specify the intake to upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.ReviewFileLocation, t)

	// specify the app versions and the platform information
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9.5", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.2.3", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should be an intake layout for a specified review layout
	layout, layoutId, err := testData.DataApi.IntakeLayoutForReviewLayoutVersion(1, 0, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, true, layoutId > 0)
	test.Equals(t, true, layout != nil)

	// ... and a review layout for a specified intake layout
	layout, layoutId, err = testData.DataApi.ReviewLayoutForIntakeLayoutVersion(1, 0, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, true, layoutId > 0)
	test.Equals(t, true, layout != nil)

	// and an intake layout for the future app versions
	layout, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutId > 0)

	layout, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 0, Patch: 0}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutId > 0)

	layout, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 9, Patch: 6}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutId > 0)

	layout, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 15, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutId > 0)

	// there should be no layout for a version prior to 0.9.5
	layout, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 8, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.Equals(t, api.NoRowsError, err)

	// now lets go ahead and apply another major upgrade to version 3.0 of the patient and doctor apps

	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-3-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-3-0-0.json", test_integration.ReviewFileLocation, t)

	// specify the patient app version that will support the major upgrade
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9.7", t)

	// specify the doctor app version that will support the major upgrade for the review
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "2.1.0", t)

	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	err = writer.Close()
	test.OK(t, err)

	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should be an active version of each type of layout for the major version,
	// pertaining to the different major versions of the app
	var count int64
	err = testData.DB.QueryRow(`select count(*) from layout_version where layout_purpose = ? and status = 'ACTIVE' and major = 2`, api.ReviewPurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	err = testData.DB.QueryRow(`select count(*) from layout_version where layout_purpose = ? and status = 'ACTIVE' and major = 2`, api.ConditionIntakePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets get the layoutVersionIds to ensure that we are getting back the right layout for the right version of the app
	var v1ReviewLayoutVersionID, v2ReviewLayoutVersionID, v1IntakeLayoutVersionID, v2IntakeLayoutVersionID int64
	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and major = 2 and minor = 0 and patch = 0 and layout_purpose = ?`, api.ConditionIntakePurpose).Scan(&v1IntakeLayoutVersionID)
	test.OK(t, err)

	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and major = 2 and minor = 0 and patch = 0 and layout_purpose = ?`, api.ReviewPurpose).Scan(&v1ReviewLayoutVersionID)
	test.OK(t, err)

	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and major = 3 and minor = 0 and patch = 0 and layout_purpose = ?`, api.ConditionIntakePurpose).Scan(&v2IntakeLayoutVersionID)
	test.OK(t, err)

	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and major = 3 and minor = 0 and patch = 0 and layout_purpose = ?`, api.ReviewPurpose).Scan(&v2ReviewLayoutVersionID)
	test.OK(t, err)

	_, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 1, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, v1IntakeLayoutVersionID, layoutId)

	// patient version 1.9.6 should return the version 2.0 instead of 3.0
	_, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 1, Minor: 9, Patch: 6}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, v1IntakeLayoutVersionID, layoutId)

	_, layoutId, err = testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, v2IntakeLayoutVersionID, layoutId)

	layout, layoutId, err = testData.DataApi.ReviewLayoutForIntakeLayoutVersion(2, 0, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, v1ReviewLayoutVersionID, layoutId)

	layout, layoutId, err = testData.DataApi.ReviewLayoutForIntakeLayoutVersion(3, 0, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, v2ReviewLayoutVersionID, layoutId)

}

func TestLayoutVersioning_MinorUpgrade(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// need to first do a major upgrade to be able to test minor upgrades
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.ReviewFileLocation, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)
	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// ensure that minor upgrades are not possible when just 1 version is specified
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-0.json", test_integration.IntakeFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// now do a minor upgrade
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-1-0.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point, there should be just 1 active layout version for the given minor version
	var count int64
	err = testData.DB.QueryRow(`select count(*) from layout_version where major = 2 and status = 'ACTIVE' and layout_purpose =?`, api.ConditionIntakePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)
	err = testData.DB.QueryRow(`select count(*) from layout_version where major = 2 and status = 'ACTIVE' and layout_purpose =?`, api.ReviewPurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets get the layoutVersionId of the minor version upgrades to ensure that
	// it is now the latest version that we return to the client
	var upgradedIntakeLayoutVersionID, upgradedReviewLayoutVersionID int64
	err = testData.DB.QueryRow(`select id from layout_version where major = 2 and minor = 1 and patch = 0 and layout_purpose = ?`, api.ConditionIntakePurpose).Scan(&upgradedIntakeLayoutVersionID)
	test.OK(t, err)
	err = testData.DB.QueryRow(`select id from layout_version where major = 2 and minor = 1 and patch = 0 and layout_purpose = ?`, api.ReviewPurpose).Scan(&upgradedReviewLayoutVersionID)
	test.OK(t, err)

	_, layoutId, err := testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, upgradedIntakeLayoutVersionID, layoutId)

	_, layoutId, err = testData.DataApi.ReviewLayoutForIntakeLayoutVersion(2, 1, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, upgradedReviewLayoutVersionID, layoutId)
}

func TestLayoutVersioning_IncompatiblePatchUpgrades(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// need to first do a major upgrade to be able to test minor upgrades
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.ReviewFileLocation, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)
	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now attempt to do a patch upgrade for review and intake,
	// but have the changes in the file represent upgrades that are incompatible with the previous version
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-1.json", "../../info_intake/intake-minor-test.json", t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-1.json", "../../info_intake/review-minor-test.json", t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// however, running the same incompatible patch upgrades as a minor upgrade should work
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-1.json", "../../info_intake/intake-minor-test.json", t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-1-1.json", "../../info_intake/review-minor-test.json", t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestLayoutVersioning_PatchUpgrade(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// need to first do a major upgrade to be able to test minor upgrades
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.ReviewFileLocation, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.9", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)
	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now do a patch upgrade
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-1.json", test_integration.IntakeFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point ensure that there is just 1 active version for the condition intake
	var count int64
	err = testData.DB.QueryRow(`select count(*) from layout_version where status = 'ACTIVE' and layout_purpose = ? and major = 2`, api.ConditionIntakePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// get the layoutVersionID of the patched upgrade
	var patchedIntakeLayoutVersionID int64
	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and layout_purpose = ? and major = 2 and minor = 0 and patch = 1`, api.ConditionIntakePurpose).Scan(&patchedIntakeLayoutVersionID)
	test.OK(t, err)

	// ensure that the latet version being returned to a client is now the patched version
	_, layoutId, err := testData.DataApi.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	test.OK(t, err)
	test.Equals(t, patchedIntakeLayoutVersionID, layoutId)

	// now do a patched upgrade of the review
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-1.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point ensure that there is just 1 active version for the condition intake
	err = testData.DB.QueryRow(`select count(*) from layout_version where status = 'ACTIVE' and layout_purpose = ? and major = 2`, api.ConditionIntakePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// get the layoutVersionID of the patched upgrade
	var patchedReviewLayoutVersionID int64
	err = testData.DB.QueryRow(`select id from layout_version where status = 'ACTIVE' and layout_purpose = ? and major = 2 and minor = 0 and patch = 1`, api.ReviewPurpose).Scan(&patchedReviewLayoutVersionID)
	test.OK(t, err)

	// ensure that the version returned for the provided intake version is the latest patch version of the review
	_, layoutId, err = testData.DataApi.ReviewLayoutForIntakeLayoutVersion(2, 0, apiservice.HEALTH_CONDITION_ACNE_ID)
	test.OK(t, err)
	test.Equals(t, patchedReviewLayoutVersionID, layoutId)

	// now ensure that we can do patched version upgrade of both layouts at once
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-5.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-5.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)

	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestLayoutVersioning_DiagnosisLayout(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// ensure that we can successfully upload a diagnosis layout by itself
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "diagnose", "diagnose-2-0-0.json", test_integration.DiagnosisFileLocation, t)
	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// there should be 1 valid diagnosis layout for the major version
	var count int64
	err = testData.DB.QueryRow(`select count(*) from layout_version where layout_purpose = ? and status = 'ACTIVE' and major = 2`, api.DiagnosePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// should be able to upload patch and minor versions of the diagnosis no problem
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "diagnose", "diagnose-2-1-0.json", test_integration.DiagnosisFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// patch version
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "diagnose", "diagnose-2-1-1.json", test_integration.DiagnosisFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// should still have just 1 active version
	err = testData.DB.QueryRow(`select count(*) from layout_version where layout_purpose = ? and status = 'ACTIVE' and major = 2`, api.DiagnosePurpose).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)
}

func TestLayoutVersioning_MajorUpgradeValidation(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// ensure that a major upgrade requires both layouts to be present
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)

	err := writer.Close()
	test.OK(t, err)

	admin := test_integration.CreateRandomAdmin(t, testData)
	resp, err := testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// ensure that major upgrades require app versions to be present
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.IntakeFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// ensure that major upgrades requires the platform to be present
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "2.0.0", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "2.0.0", t)

	err = writer.Close()
	test.OK(t, err)

	resp, err = testData.AuthPost(testData.APIServer.URL+router.LayoutUploadURLPath, writer.FormDataContentType(), body, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)
}
