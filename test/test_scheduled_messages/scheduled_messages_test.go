package test_scheduled_messages

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestScheduledMessageDeactivateForPatient(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)

	pvr := test_integration.CreateRandomPatientVisitInState("CA", t, testData)
	patientVisit, err := testData.DataAPI.GetPatientVisitFromID(pvr.PatientVisitID)
	test.OK(t, err)

	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMScheduled)
	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMScheduled)
	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMScheduled)
	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMSent)
	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMError)
	insertScheduledMessage(t, testData, patientVisit.PatientID.Int64(), common.SMProcessing)

	aff, err := testData.DataAPI.DeactivateScheduledMessagesForPatient(patientVisit.PatientID.Int64())
	test.OK(t, err)
	test.Equals(t, int64(3), aff)
}

type testTyped struct{}

func (t testTyped) TypeName() string { return "test" }

func insertScheduledMessage(t *testing.T, testData *test_integration.TestData, patientID int64, status common.ScheduledMessageStatus) {
	_, err := testData.DataAPI.CreateScheduledMessage(&common.ScheduledMessage{
		Event:     "test_event",
		PatientID: patientID,
		Message:   testTyped{},
		Scheduled: time.Now(),
		Status:    status,
	})
	test.OK(t, err)
}
