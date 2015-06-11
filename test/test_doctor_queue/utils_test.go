package test_doctor_queue

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func getExpiresTimeFromDoctorForCase(testData *test_integration.TestData, t *testing.T, patientCaseID int64) *time.Time {
	doctorAssignments, err := testData.DataAPI.GetDoctorsAssignedToPatientCase(patientCaseID)
	test.OK(t, err)
	return doctorAssignments[0].Expires
}

func getUnclaimedItemsForDoctor(doctorID int64, t *testing.T, testData *test_integration.TestData) []*api.DoctorQueueItem {
	unclaimedItems, err := testData.DataAPI.GetElligibleItemsInUnclaimedQueue(doctorID)
	test.OK(t, err)
	return unclaimedItems
}
