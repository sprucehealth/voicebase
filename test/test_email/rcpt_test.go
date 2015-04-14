package test_email

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestRecipients(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pAccountID := test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()

	rcpt, err := testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", false)
	test.OK(t, err)
	if len(rcpt) != 1 {
		t.Fatalf("Expected 1 recipient, not %d", len(rcpt))
	} else if rcpt[0].Name != "Test Test" {
		t.Fatalf("Unexpected recipient name '%s'", rcpt[0].Name)
	} else if !strings.HasSuffix(rcpt[0].Email, "@example.com") {
		t.Fatalf("Unexpected recipient email '%s'", rcpt[0].Email)
	} else if rcpt[0].AccountID != pAccountID {
		t.Fatalf("Expected account ID %d, got %d", pAccountID, rcpt[0].AccountID)
	}

	rcpt, err = testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", true)
	test.OK(t, err)
	if len(rcpt) != 1 {
		t.Fatalf("Expected 1 recipient, not %d", len(rcpt))
	}

	rcpt, err = testData.DataAPI.EmailRecipients([]int64{pAccountID})
	test.OK(t, err)
	if len(rcpt) != 1 {
		t.Fatalf("Expected 1 recipient, not %d", len(rcpt))
	}
}

func TestOptout(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pAccountID := test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()

	test.OK(t, testData.DataAPI.EmailUpdateOptOut(pAccountID, "blah", true))
	rcpt, err := testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", false)
	test.OK(t, err)
	if len(rcpt) != 0 {
		t.Fatalf("Expected 0 recipients, not %d", len(rcpt))
	}
	rcpt, err = testData.DataAPI.EmailRecipients([]int64{pAccountID})
	test.OK(t, err)
	if len(rcpt) != 1 {
		t.Fatalf("Expected 1 recipient, not %d", len(rcpt))
	}

	test.OK(t, testData.DataAPI.EmailUpdateOptOut(pAccountID, "blah", false))
	rcpt, err = testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", false)
	test.OK(t, err)
	if len(rcpt) != 1 {
		t.Fatalf("Expected 1 recipient, not %d", len(rcpt))
	}

	test.OK(t, testData.DataAPI.EmailUpdateOptOut(pAccountID, "all", true))
	rcpt, err = testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", false)
	test.OK(t, err)
	if len(rcpt) != 0 {
		t.Fatalf("Expected 0 recipients, not %d", len(rcpt))
	}
}

func TestOnlyOnce(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pAccountID := test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()

	test.OK(t, testData.DataAPI.EmailRecordSend([]int64{pAccountID}, "blah"))
	rcpt, err := testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", true)
	test.OK(t, err)
	if len(rcpt) != 0 {
		t.Fatalf("Expected 0 recipients, not %d", len(rcpt))
	}

	// Test multiple entries in sent table
	test.OK(t, testData.DataAPI.EmailRecordSend([]int64{pAccountID}, "blah"))
	rcpt, err = testData.DataAPI.EmailRecipientsWithOptOut([]int64{pAccountID}, "blah", true)
	test.OK(t, err)
	if len(rcpt) != 0 {
		t.Fatalf("Expected 0 recipients, not %d", len(rcpt))
	}
}
