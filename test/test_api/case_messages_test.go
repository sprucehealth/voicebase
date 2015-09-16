package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestCaseMessageRead(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	_, err := testData.DataAPI.CreateSKU(&common.SKU{
		Type:         "acne",
		CategoryType: common.SCVisit,
	})

	accountID, err := testData.AuthAPI.CreateAccount("test@patient.com", "abc", api.RolePatient)
	test.OK(t, err)
	p := &common.Patient{
		AccountID:      encoding.DeprecatedNewObjectID(accountID),
		PatientAddress: &common.Address{},
	}
	err = testData.DataAPI.RegisterPatient(p)
	test.OK(t, err)

	_, err = testData.DB.Exec(`
		INSERT INTO patient_case (patient_id, status, name, clinical_pathway_id)
		VALUES (?, ?, ?, ?)`, p.ID, common.PCStatusActive.String(), "Case Name", 1)
	test.OK(t, err)

	personID, err := testData.DataAPI.GetPersonIDByRole(api.RolePatient, p.ID.Int64())
	test.OK(t, err)

	doctorAccountID, err := testData.AuthAPI.CreateAccount("dr@sprucehealth.com", "abc", api.RoleDoctor)
	test.OK(t, err)

	doctor := &common.Doctor{
		AccountID: encoding.DeprecatedNewObjectID(doctorAccountID),
		Address:   &common.Address{},
	}
	did, err := testData.DataAPI.RegisterProvider(doctor, api.RoleDoctor)
	test.OK(t, err)

	doctorPersonID, err := testData.DataAPI.GetPersonIDByRole(api.RoleDoctor, did)
	test.OK(t, err)

	message := &common.CaseMessage{
		CaseID:   1,
		PersonID: doctorPersonID,
		Body:     "SUP",
	}

	messageID, err := testData.DataAPI.CreateCaseMessage(message)
	test.OK(t, err)

	isMsgRead, err := testData.DataAPI.IsCaseMessageRead(messageID, personID)
	test.OK(t, err)
	test.Equals(t, false, isMsgRead)

	test.OK(t, testData.DataAPI.CaseMessagesRead([]int64{messageID}, personID))

	isMsgRead, err = testData.DataAPI.IsCaseMessageRead(messageID, personID)
	test.OK(t, err)
	test.Equals(t, true, isMsgRead)

}
