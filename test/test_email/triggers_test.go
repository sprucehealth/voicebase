package test_email

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTriggers(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// Sanity check
	if msgs := testData.EmailService.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}

	test.OK(t, testData.Config.Cfg.Update(map[string]interface{}{
		"Email.Campaign.Welcome.Enabled": false,
	}))

	test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()

	if msgs := testData.EmailService.Reset(); len(msgs) != 0 {
		t.Error("welcome email should not have been when disabled")
	}

	test.OK(t, testData.Config.Cfg.Update(map[string]interface{}{
		"Email.Campaign.Welcome.Enabled": true,
	}))

	test_integration.SignupRandomTestPatient(t, testData).Patient.AccountID.Int64()

	if msgs := testData.EmailService.Reset(); len(msgs) != 1 {
		t.Error("welcome email not sent")
	} else if m := msgs[0]; m.Type != "welcome" {
		t.Errorf("welcome email should be of type 'welcome' not '%s'", m.Type)
	}
}
