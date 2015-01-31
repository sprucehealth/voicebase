package test_intake

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
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

	pathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should be an intake layout for a specified review layout
	layout, layoutID, err := testData.DataAPI.IntakeLayoutForReviewLayoutVersion(1, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutID > 0)
	test.Equals(t, true, layout != nil)

	// ... and a review layout for a specified intake layout
	layout, layoutID, err = testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(1, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutID > 0)
	test.Equals(t, true, layout != nil)

	// and an intake layout for the future app versions
	layout, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutID > 0)

	layout, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 0, Patch: 0}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutID > 0)

	layout, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 9, Patch: 6}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutID > 0)

	layout, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 15, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layout != nil)
	test.Equals(t, true, layoutID > 0)

	// there should be no layout for a version prior to 0.9.5
	layout, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 0, Minor: 8, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.Equals(t, true, api.IsErrNotFound(err))

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

	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	_, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 1, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, v1IntakeLayoutVersionID, layoutID)

	// patient version 1.9.6 should return the version 2.0 instead of 3.0
	_, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 1, Minor: 9, Patch: 6}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, v1IntakeLayoutVersionID, layoutID)

	_, layoutID, err = testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, v2IntakeLayoutVersionID, layoutID)

	layout, layoutID, err = testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(2, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, v1ReviewLayoutVersionID, layoutID)

	layout, layoutID, err = testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(3, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, v2ReviewLayoutVersionID, layoutID)

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

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// ensure that minor upgrades are not possible when just 1 version is specified
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-0.json", test_integration.IntakeFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	pathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)

	_, layoutID, err := testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, upgradedIntakeLayoutVersionID, layoutID)

	_, layoutID, err = testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(2, 1, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, upgradedReviewLayoutVersionID, layoutID)
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

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now attempt to do a patch upgrade for review and intake,
	// but have the changes in the file represent upgrades that are incompatible with the previous version
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-1.json", "../../info_intake/minor-intake-test.json", t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-1.json", "../../info_intake/minor-review-test.json", t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	// however, running the same incompatible patch upgrades as a minor upgrade should work
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-1.json", "../../info_intake/minor-intake-test.json", t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-1-1.json", "../../info_intake/minor-review-test.json", t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// now do a patch upgrade
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-1.json", test_integration.IntakeFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	pathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)

	// ensure that the latet version being returned to a client is now the patched version
	_, layoutID, err := testData.DataAPI.IntakeLayoutForAppVersion(&common.Version{Major: 2, Minor: 9, Patch: 5}, common.IOS,
		pathway.ID, api.EN_LANGUAGE_ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, patchedIntakeLayoutVersionID, layoutID)

	// now do a patched upgrade of the review
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-1.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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
	_, layoutID, err = testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(2, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, patchedReviewLayoutVersionID, layoutID)

	// now ensure that we can do patched version upgrade of both layouts at once
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-5.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-5.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)

	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// patch version
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "diagnose", "diagnose-2-1-1.json", test_integration.DiagnosisFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
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

	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLayoutVersioning_FollowupSupport(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// add a followup layout and ensure that both followup and new-visit layouts stay active
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-0-0.json", test_integration.FollowupIntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-0-0.json", test_integration.FollowupReviewFileLocation, t)

	// specify the app versions and the platform information
	test_integration.AddFieldToMultipartWriter(writer, "patient_app_version", "1.9.5", t)
	test_integration.AddFieldToMultipartWriter(writer, "doctor_app_version", "1.2.3", t)
	test_integration.AddFieldToMultipartWriter(writer, "platform", "iOS", t)

	err := writer.Close()
	test.OK(t, err)

	pathway, err := testData.DataAPI.PathwayForTag(api.AcnePathwayTag, api.PONone)
	test.OK(t, err)

	resp, err := testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should be active layouts for a new acne visit
	layout, layoutId1a, err := testData.DataAPI.IntakeLayoutForReviewLayoutVersion(1, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutId1a > 0)
	test.Equals(t, true, layout != nil)
	layout, layoutId1b, err := testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(1, 0, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutId1b > 0)
	test.Equals(t, true, layout != nil)

	// ... and followup
	layout, layoutID2a, err := testData.DataAPI.IntakeLayoutForReviewLayoutVersion(2, 0, pathway.ID, test_integration.SKUAcneFollowup)
	test.OK(t, err)
	test.Equals(t, true, layoutID2a > 0)
	test.Equals(t, true, layoutId1a != layoutID2a)
	test.Equals(t, true, layout != nil)
	layout, layoutId2b, err := testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(2, 0, pathway.ID, test_integration.SKUAcneFollowup)
	test.OK(t, err)
	test.Equals(t, true, layoutId2b > 0)
	test.Equals(t, true, layoutId1b != layoutId2b)
	test.Equals(t, true, layout != nil)

	// now lets do a minor version upgrade for followup and ensure that all worked well
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-2-1-0.json", test_integration.FollowupIntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-2-1-0.json", test_integration.FollowupReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should be just 1 active followup pair
	var count int64
	err = testData.DB.QueryRow(`select count(*) from layout_version inner join sku on sku.id = sku_id where major = 1 and status = 'ACTIVE' and layout_purpose =? and sku.type  = ?`, api.ConditionIntakePurpose, test_integration.SKUAcneFollowup).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)
	err = testData.DB.QueryRow(`select count(*) from layout_version inner join sku on sku.id = sku_id where major = 1 and status = 'ACTIVE' and layout_purpose = ? and sku.type = ?`, api.ReviewPurpose, test_integration.SKUAcneFollowup).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// and 1 active pair for the intake
	err = testData.DB.QueryRow(`select count(*) from layout_version inner join sku on sku.id = sku_id where major = 1 and status = 'ACTIVE' and layout_purpose =? and sku.type  = ?`, api.ConditionIntakePurpose, test_integration.SKUAcneVisit).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)
	err = testData.DB.QueryRow(`select count(*) from layout_version inner join sku on sku.id = sku_id where major = 1 and status = 'ACTIVE' and layout_purpose = ? and sku.type = ?`, api.ReviewPurpose, test_integration.SKUAcneVisit).Scan(&count)
	test.OK(t, err)
	test.Equals(t, int64(1), count)

	// lets also do a minor version upgrade of the new-visit pair
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	test_integration.AddFileToMultipartWriter(writer, "intake", "intake-1-1-0.json", test_integration.IntakeFileLocation, t)
	test_integration.AddFileToMultipartWriter(writer, "review", "review-1-1-0.json", test_integration.ReviewFileLocation, t)
	err = writer.Close()
	test.OK(t, err)
	resp, err = testData.AdminAuthPost(testData.AdminAPIServer.URL+`/admin/api/layout`, writer.FormDataContentType(), body, testData.AdminUser)
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// at this point there should still be active versions for the followup and intake pairs
	layout, layoutId3a, err := testData.DataAPI.IntakeLayoutForReviewLayoutVersion(1, 1, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutId3a > 0)
	test.Equals(t, true, layoutId3a != layoutId1a)
	test.Equals(t, true, layout != nil)
	layout, layoutId3b, err := testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(1, 1, pathway.ID, test_integration.SKUAcneVisit)
	test.OK(t, err)
	test.Equals(t, true, layoutId3b > 0)
	test.Equals(t, true, layoutId3b != layoutId1b)
	test.Equals(t, true, layout != nil)

	layout, layoutId4a, err := testData.DataAPI.IntakeLayoutForReviewLayoutVersion(2, 1, pathway.ID, test_integration.SKUAcneFollowup)
	test.OK(t, err)
	test.Equals(t, true, layoutId4a > 0)
	test.Equals(t, true, layoutId4a != layoutId3a)
	test.Equals(t, true, layoutId4a != layoutID2a)
	test.Equals(t, true, layout != nil)
	layout, layoutId4b, err := testData.DataAPI.ReviewLayoutForIntakeLayoutVersion(2, 1, pathway.ID, test_integration.SKUAcneFollowup)
	test.OK(t, err)
	test.Equals(t, true, layoutId4b > 0)
	test.Equals(t, true, layoutId4b != layoutId3b)
	test.Equals(t, true, layoutId4b != layoutId2b)
	test.Equals(t, true, layout != nil)
}

const (
	Intake   = "CONDITION_INTAKE"
	Review   = "REVIEW"
	Diagnose = "DIAGNOSE"
)

func insertLayoutBlob(t *testing.T, testData *test_integration.TestData, blob string) (int64, error) {
	res, err := testData.DB.Exec(
		`INSERT INTO layout_blob_storage (layout) VALUES (CAST(? AS BINARY))`, blob)
	test.OK(t, err)
	return res.LastInsertId()
}

func insertClinicalPathway(t *testing.T, testData *test_integration.TestData, tag string) (int64, error) {
	res, err := testData.DB.Exec(
		`INSERT INTO clinical_pathway (tag, name, medicine_branch, status)
			VALUES (?, ?, ?, 'ACTIVE')`, tag, tag, tag)
	test.OK(t, err)
	return res.LastInsertId()
}

func insertLayoutVersion(t *testing.T, testData *test_integration.TestData, purpose string, clinicalPathwayID, blobID, major, minor, patch int64) (int64, error) {
	res, err := testData.DB.Exec(
		`INSERT INTO layout_version (clinical_pathway_id, status, role, layout_purpose, layout_blob_storage_id, major, minor, patch)
			VALUES (?, 'ACTIVE', 'PATIENT', ?, ?, ?, ?, ?)`, clinicalPathwayID, purpose, blobID, major, minor, patch)
	test.OK(t, err)
	return res.LastInsertId()
}

func TestLayoutVersionMappingDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	blobID, err := insertLayoutBlob(t, testData, "{Blob}")
	test.OK(t, err)
	cpID1, err := insertClinicalPathway(t, testData, "pathway_tag")
	test.OK(t, err)
	cpID2, err := insertClinicalPathway(t, testData, "pathway_tag2")
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Intake, cpID1, blobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Intake, cpID1, blobID, 1, 0, 1)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Review, cpID1, blobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Review, cpID1, blobID, 1, 0, 2)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Diagnose, cpID1, blobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Diagnose, cpID1, blobID, 1, 0, 1)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Intake, cpID2, blobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Intake, cpID2, blobID, 1, 0, 1)
	test.OK(t, err)

	mappings, err := testData.DataAPI.LayoutVersionMapping()
	test.OK(t, err)
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 0}, mappings["pathway_tag"][Intake][0])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 1}, mappings["pathway_tag"][Intake][1])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 0}, mappings["pathway_tag"][Review][0])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 2}, mappings["pathway_tag"][Review][1])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 0}, mappings["pathway_tag"][Diagnose][0])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 1}, mappings["pathway_tag"][Diagnose][1])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 0}, mappings["pathway_tag2"][Intake][0])
	test.Equals(t, &common.Version{Major: 1, Minor: 0, Patch: 1}, mappings["pathway_tag2"][Intake][1])
}

func TestLayoutTemplateDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	iblobID, err := insertLayoutBlob(t, testData, "{iBlob}")
	test.OK(t, err)
	rblobID, err := insertLayoutBlob(t, testData, "{rBlob}")
	test.OK(t, err)
	dblobID, err := insertLayoutBlob(t, testData, "{dBlob}")
	test.OK(t, err)
	cpID1, err := insertClinicalPathway(t, testData, "pathway_tag")
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Intake, cpID1, iblobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Review, cpID1, rblobID, 1, 0, 0)
	test.OK(t, err)
	_, err = insertLayoutVersion(t, testData, Diagnose, cpID1, dblobID, 1, 0, 0)
	test.OK(t, err)

	template, err := testData.DataAPI.LayoutTemplate("pathway_tag", Intake, &common.Version{Major: 1, Minor: 0, Patch: 0})
	test.OK(t, err)
	test.Equals(t, "{iBlob}", string(template))
	template, err = testData.DataAPI.LayoutTemplate("pathway_tag", Review, &common.Version{Major: 1, Minor: 0, Patch: 0})
	test.OK(t, err)
	test.Equals(t, "{rBlob}", string(template))
	template, err = testData.DataAPI.LayoutTemplate("pathway_tag", Diagnose, &common.Version{Major: 1, Minor: 0, Patch: 0})
	test.OK(t, err)
	test.Equals(t, "{dBlob}", string(template))
}
