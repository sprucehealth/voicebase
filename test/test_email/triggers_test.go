package test_email

import (
	"testing"

	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTriggers(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Sanity check
	if msgs := testData.EmailService.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}

	pAccountID := test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()
	_ = pAccountID

	if msgs := testData.EmailService.Reset(); len(msgs) != 1 {
		t.Error("welcome email not sent")
	} else if m := msgs[0]; m.Type != "welcome" {
		t.Errorf("welcome email should be of type 'welcome' not '%s'", m.Type)
	}
}
