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
	testNotifyTypes[patient_case.CNMessage] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&testMessageData{})).Interface())
	testNotifyTypes[patient_case.CNTreatmentPlan] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&testTreatmentPlanData{})).Interface())
	testNotifyTypes[patient_case.CNVisitSubmitted] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&testVisitSubmittedNotification{})).Interface())
	testNotifyTypes[patient_case.CNIncompleteVisit] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&testIncompleteVisitNotification{})).Interface())
	return testNotifyTypes
}
