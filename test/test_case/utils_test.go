package test_case

import (
	"reflect"

	"github.com/sprucehealth/backend/patient_case"
)

type testTreatmentPlanData struct {
}

func (t *testTreatmentPlanData) TypeName() string {
	return patient_case.CNTreatmentPlan
}

type testMessageData struct {
}

func (t *testMessageData) TypeName() string {
	return patient_case.CNMessage
}

type testVisitSubmittedNotification struct {
}

func (t *testVisitSubmittedNotification) TypeName() string {
	return patient_case.CNVisitSubmitted
}

type testIncompleteVisitNotification struct {
}

func (t *testIncompleteVisitNotification) TypeName() string {
	return patient_case.CNIncompleteVisit
}

func getNotificationTypes() map[string]reflect.Type {
	testNotifyTypes := make(map[string]reflect.Type)
	testNotifyTypes[(new(testMessageData)).TypeName()] = reflect.TypeOf(testMessageData{})
	testNotifyTypes[(new(testTreatmentPlanData)).TypeName()] = reflect.TypeOf(testTreatmentPlanData{})
	testNotifyTypes[(new(testVisitSubmittedNotification)).TypeName()] = reflect.TypeOf(testVisitSubmittedNotification{})
	testNotifyTypes[(new(testIncompleteVisitNotification)).TypeName()] = reflect.TypeOf(testIncompleteVisitNotification{})
	return testNotifyTypes
}
