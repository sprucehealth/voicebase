package test_api

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestBanking(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	accountID, err := testData.AuthAPI.CreateAccount("test+perms@sprucehealth.com", "xyz", api.RoleAdmin)
	test.OK(t, err)

	bankAccount := &common.BankAccount{
		AccountID:         accountID,
		StripeRecipientID: "123",
		Default:           true,
		VerifyAmount1:     123,
		VerifyAmount2:     999,
		VerifyTransfer1ID: "12345",
		VerifyTransfer2ID: "54321",
		VerifyExpires:     time.Date(2015, 1, 2, 3, 4, 5, 0, time.UTC),
		Verified:          false,
	}
	bankAccountID, err := testData.DataAPI.AddBankAccount(bankAccount)
	test.OK(t, err)

	bankAccounts, err := testData.DataAPI.ListBankAccounts(accountID)
	test.OK(t, err)
	test.Equals(t, 1, len(bankAccounts))
	test.Equals(t, bankAccountID, bankAccounts[0].ID)
	test.Equals(t, accountID, bankAccounts[0].AccountID)
	test.Equals(t, bankAccount.StripeRecipientID, bankAccounts[0].StripeRecipientID)
	test.Equals(t, bankAccount.Default, bankAccounts[0].Default)
	test.Equals(t, bankAccount.VerifyAmount1, bankAccounts[0].VerifyAmount1)
	test.Equals(t, bankAccount.VerifyAmount2, bankAccounts[0].VerifyAmount2)
	test.Equals(t, bankAccount.VerifyTransfer1ID, bankAccounts[0].VerifyTransfer1ID)
	test.Equals(t, bankAccount.VerifyTransfer2ID, bankAccounts[0].VerifyTransfer2ID)
	test.Equals(t, bankAccount.VerifyExpires.Unix(), bankAccounts[0].VerifyExpires.Unix())
	test.Equals(t, bankAccount.Verified, bankAccounts[0].Verified)

	n, err := testData.DataAPI.UpdateBankAccount(bankAccountID, &api.BankAccountUpdate{
		VerifyAmount1:     ptr.Int(0),
		VerifyAmount2:     ptr.Int(0),
		VerifyTransfer1ID: ptr.String(""),
		VerifyTransfer2ID: ptr.String(""),
		VerifyExpires:     ptr.Time(time.Time{}),
		Verified:          ptr.Bool(true),
	})
	test.OK(t, err)
	test.Equals(t, 1, n)

	bankAccounts, err = testData.DataAPI.ListBankAccounts(accountID)
	test.OK(t, err)
	test.Equals(t, 1, len(bankAccounts))
	test.Equals(t, 0, bankAccounts[0].VerifyAmount1)
	test.Equals(t, 0, bankAccounts[0].VerifyAmount2)
	test.Equals(t, "", bankAccounts[0].VerifyTransfer1ID)
	test.Equals(t, "", bankAccounts[0].VerifyTransfer2ID)
	test.Assert(t, bankAccounts[0].VerifyExpires.IsZero(), "Update VerifyExpires should be zero")
	test.Equals(t, true, bankAccounts[0].Verified)
}
