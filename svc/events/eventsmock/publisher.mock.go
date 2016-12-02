// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/sprucehealth/backend/svc/events (interfaces: Publisher)

package eventsmock

import (
	gomock "github.com/golang/mock/gomock"
	events "github.com/sprucehealth/backend/svc/events"
)

// Mock of Publisher interface
type MockPublisher struct {
	ctrl     *gomock.Controller
	recorder *_MockPublisherRecorder
}

// Recorder for MockPublisher (not exported)
type _MockPublisherRecorder struct {
	mock *MockPublisher
}

func NewMockPublisher(ctrl *gomock.Controller) *MockPublisher {
	mock := &MockPublisher{ctrl: ctrl}
	mock.recorder = &_MockPublisherRecorder{mock}
	return mock
}

func (_m *MockPublisher) EXPECT() *_MockPublisherRecorder {
	return _m.recorder
}

func (_m *MockPublisher) Publish(_param0 events.Marshaler) error {
	ret := _m.ctrl.Call(_m, "Publish", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockPublisherRecorder) Publish(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Publish", arg0)
}

func (_m *MockPublisher) PublishAsync(_param0 events.Marshaler) {
	_m.ctrl.Call(_m, "PublishAsync", _param0)
}

func (_mr *_MockPublisherRecorder) PublishAsync(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PublishAsync", arg0)
}
