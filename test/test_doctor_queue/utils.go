package test_doctor_queue

import (
	"carefront/api"
	"carefront/test/test_integration"
	"testing"
	"time"
)

func getExpiresTimeFromDoctorForCase(testData *test_integration.TestData, t *testing.T, patientCaseId int64) *time.Time {
	doctorAssignments, err := testData.DataApi.GetDoctorsAssignedToPatientCase(patientCaseId)
	if err != nil {
		t.Fatal(err)
	}
	return doctorAssignments[0].Expires
}

func getUnclaimedItemsForDoctor(doctorId int64, t *testing.T, testData *test_integration.TestData) []*api.DoctorQueueItem {
	unclaimedItems, err := testData.DataApi.GetElligibleItemsInUnclaimedQueue(doctorId)
	if err != nil {
		t.Fatal(err)
	}
	return unclaimedItems
}
