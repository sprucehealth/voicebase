package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestParentalConsent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("patient@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID.Int64()

	accountID, err = testData.AuthAPI.CreateAccount("parent@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient = &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	parentPatientID := patient.ID.Int64()

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, false, patient.HasParentalConsent)

	consent, err := testData.DataAPI.ParentalConsent(parentPatientID, patientID)
	test.Assert(t, err != nil, "Expected error for non-existant link between parent and child")
	test.Assert(t, api.IsErrNotFound(err), "Expected IsErrNotFound")
	test.Equals(t, (*common.ParentalConsent)(nil), consent)

	test.OK(t, testData.DataAPI.GrantParentChildConsent(parentPatientID, patientID, "likely-just-a-friend"))

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, false, patient.HasParentalConsent)

	consent, err = testData.DataAPI.ParentalConsent(parentPatientID, patientID)
	test.OK(t, err)
	test.Equals(t, true, consent.Consented)

	test.OK(t, testData.DataAPI.ParentalConsentCompletedForPatient(patientID))

	patient, err = testData.DataAPI.Patient(patientID, true)
	test.OK(t, err)
	test.Equals(t, true, patient.HasParentalConsent)

	consent, err = testData.DataAPI.ParentalConsent(parentPatientID, patientID)
	test.OK(t, err)
	test.Equals(t, true, consent.Consented)
}

func TestParentalConsentProof(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	pr := test_integration.SignupRandomTestPatient(t, testData)

	governmentIDPhotoID, _ := test_integration.UploadPhoto(t, testData, pr.Patient.AccountID.Int64())
	selfiePhotoID, _ := test_integration.UploadPhoto(t, testData, pr.Patient.AccountID.Int64())

	rowsAffected, err := testData.DataAPI.UpsertParentConsentProof(
		pr.Patient.ID.Int64(),
		&api.ParentalConsentProof{
			GovernmentIDPhotoID: ptr.Int64(governmentIDPhotoID),
		})
	test.OK(t, err)
	test.Equals(t, int64(1), rowsAffected)

	// check if the proof was inserted as expected
	proof, err := testData.DataAPI.ParentConsentProof(pr.Patient.ID.Int64())
	test.OK(t, err)
	test.Equals(t, governmentIDPhotoID, *proof.GovernmentIDPhotoID)
	test.Equals(t, true, proof.SelfiePhotoID == nil)

	// now try to update (if rowsAffected was 2 then row was updated)
	rowsAffected, err = testData.DataAPI.UpsertParentConsentProof(
		pr.Patient.ID.Int64(), &api.ParentalConsentProof{
			SelfiePhotoID: ptr.Int64(selfiePhotoID),
		})
	test.OK(t, err)
	test.Equals(t, int64(2), rowsAffected)

	proof, err = testData.DataAPI.ParentConsentProof(pr.Patient.ID.Int64())
	test.OK(t, err)
	test.Equals(t, governmentIDPhotoID, *proof.GovernmentIDPhotoID)
	test.Equals(t, selfiePhotoID, *proof.SelfiePhotoID)
}

func TestPatientParentID(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("patient@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient := &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID.Int64()

	_, err = testData.DataAPI.PatientParentID(patientID)
	test.Assert(t, api.IsErrNotFound(err), "Expected no patient_parent record to be found")

	accountID, err = testData.AuthAPI.CreateAccount("parent@sprucehealth.com", "12345", api.RolePatient)
	test.OK(t, err)
	patient = &common.Patient{
		AccountID: encoding.NewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	parentPatientID := patient.ID.Int64()
	test.OK(t, testData.DataAPI.GrantParentChildConsent(parentPatientID, patientID, "likely-just-a-friend"))

	id, err := testData.DataAPI.PatientParentID(patientID)
	test.OK(t, err)
	test.Equals(t, parentPatientID, id)
}
