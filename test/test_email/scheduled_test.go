package test_email

import (
	"testing"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/email/campaigns"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestScheduledCampaigns(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	// Sanity check
	if msgs := testData.EmailService.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}

	p := test_integration.SignupRandomTestPatient(t, testData).Patient

	dispatcher := dispatch.New()
	testMand := &email.TestMandrill{}
	emailService := email.NewOptoutChecker(testData.DataAPI, testMand, testData.Config.Cfg, dispatcher)
	lock := &test_integration.TestLock{}

	signer, err := sig.NewSigner([][]byte{[]byte("foo")}, nil)
	test.OK(t, err)
	w := campaigns.NewWorker(testData.DataAPI, emailService, "", signer, testData.Config.Cfg, lock, metrics.NewRegistry())
	if err := w.Do(); err != nil {
		t.Fatal(err)
	}

	// Test abandoned visit campaign

	test_integration.CreatePatientVisitForPatient(p.ID, testData, t)

	if msgs := testMand.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}

	_, err = testData.DB.Exec(`DELETE FROM email_campaign_state`)
	test.OK(t, err)

	res, err := testData.DB.Exec(`UPDATE patient_visit SET creation_date = NOW() - INTERVAL 60 day WHERE status = ?`, common.PVStatusOpen)
	test.OK(t, err)
	n, err := res.RowsAffected()
	test.OK(t, err)
	test.Equals(t, int64(1), n)
	if err := w.Do(); err != nil {
		t.Fatal(err)
	}
	if msgs := testMand.Reset(); len(msgs) != 1 {
		t.Error("abandoned visit campaign email not sent")
	}

	if err := w.Do(); err != nil {
		t.Fatal(err)
	}
	if msgs := testMand.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}

	_, err = testData.DB.Exec(`DELETE FROM email_campaign_state`)
	test.OK(t, err)

	if err := w.Do(); err != nil {
		t.Fatal(err)
	}
	if msgs := testMand.Reset(); len(msgs) != 0 {
		t.Error("emails sent when none should have been")
	}
}
