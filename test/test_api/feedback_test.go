package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestFeedback(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("fdsafdsafdsa@fewffdasfdsajewlkfwe.com", "fdasfda", api.RolePatient)
	test.OK(t, err)

	patient := &common.Patient{
		AccountID: encoding.DeprecatedNewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID

	feedbackClient := feedback.NewDAL(testData.DB)

	found, err := feedbackClient.PatientFeedbackRecorded(patientID, "one")
	test.OK(t, err)
	test.Equals(t, false, found)

	cmt := "LOVED IT! EVERYTHING IS GR34t! OnE sTAR!"
	test.OK(t, feedbackClient.RecordPatientFeedback(patientID, "one", 1, &cmt, nil))

	found, err = feedbackClient.PatientFeedbackRecorded(patientID, "one")
	test.OK(t, err)
	test.Equals(t, true, found)
	found, err = feedbackClient.PatientFeedbackRecorded(patientID, "two")
	test.OK(t, err)
	test.Equals(t, false, found)

	test.OK(t, feedbackClient.RecordPatientFeedback(patientID, "two", 5, nil, nil))
	found, err = feedbackClient.PatientFeedbackRecorded(patientID, "two")
	test.OK(t, err)
	test.Equals(t, true, found)

	f, err := feedbackClient.PatientFeedback("one")
	test.OK(t, err)
	test.Equals(t, patientID, f.PatientID)
	test.Equals(t, 1, *f.Rating)
	test.Equals(t, cmt, *f.Comment)

	f, err = feedbackClient.PatientFeedback("two")
	test.OK(t, err)
	test.Equals(t, patientID, f.PatientID)
	test.Equals(t, 5, *f.Rating)
	test.Equals(t, (*string)(nil), f.Comment)
}

func TestFeedback_PendingRecord(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("fdsafdsafdsa@fewffdasfdsajewlkfwe.com", "fdasfda", api.RolePatient)
	test.OK(t, err)

	patient := &common.Patient{
		AccountID: encoding.DeprecatedNewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID

	feedbackClient := feedback.NewDAL(testData.DB)

	test.OK(t, feedbackClient.CreatePendingPatientFeedback(patientID, "case:123"))

	recorded, err := feedbackClient.PatientFeedbackRecorded(patientID, "case:123")
	test.OK(t, err)
	test.Equals(t, false, recorded)
}

func TestFeedback_Update(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("fdsafdsafdsa@fewffdasfdsajewlkfwe.com", "fdasfda", api.RolePatient)
	test.OK(t, err)

	patient := &common.Patient{
		AccountID: encoding.DeprecatedNewObjectID(accountID),
	}
	test.OK(t, testData.DataAPI.RegisterPatient(patient))
	patientID := patient.ID

	feedbackClient := feedback.NewDAL(testData.DB)

	test.OK(t, feedbackClient.CreatePendingPatientFeedback(patientID, "case:123"))

	// udpate it
	test.OK(t, feedbackClient.UpdatePatientFeedback("case:123", &feedback.PatientFeedbackUpdate{
		Dismissed: ptr.Bool(true),
	}))

	pf, err := feedbackClient.PatientFeedback("case:123")
	test.OK(t, err)
	test.Equals(t, false, pf.Pending)
	test.Equals(t, true, pf.Dismissed)
}
