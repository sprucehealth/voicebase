// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/sprucehealth/backend/svc/settings (interfaces: SettingsClient)

package settingsmock

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	settings "github.com/sprucehealth/backend/svc/settings"
	grpc "google.golang.org/grpc"
)

// Mock of SettingsClient interface
type MockSettingsClient struct {
	ctrl     *gomock.Controller
	recorder *_MockSettingsClientRecorder
}

// Recorder for MockSettingsClient (not exported)
type _MockSettingsClientRecorder struct {
	mock *MockSettingsClient
}

func NewMockSettingsClient(ctrl *gomock.Controller) *MockSettingsClient {
	mock := &MockSettingsClient{ctrl: ctrl}
	mock.recorder = &_MockSettingsClientRecorder{mock}
	return mock
}

func (_m *MockSettingsClient) EXPECT() *_MockSettingsClientRecorder {
	return _m.recorder
}

func (_m *MockSettingsClient) GetConfigs(_param0 context.Context, _param1 *settings.GetConfigsRequest, _param2 ...grpc.CallOption) (*settings.GetConfigsResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "GetConfigs", _s...)
	ret0, _ := ret[0].(*settings.GetConfigsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSettingsClientRecorder) GetConfigs(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetConfigs", _s...)
}

func (_m *MockSettingsClient) GetNodeValues(_param0 context.Context, _param1 *settings.GetNodeValuesRequest, _param2 ...grpc.CallOption) (*settings.GetNodeValuesResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "GetNodeValues", _s...)
	ret0, _ := ret[0].(*settings.GetNodeValuesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSettingsClientRecorder) GetNodeValues(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetNodeValues", _s...)
}

func (_m *MockSettingsClient) GetValues(_param0 context.Context, _param1 *settings.GetValuesRequest, _param2 ...grpc.CallOption) (*settings.GetValuesResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "GetValues", _s...)
	ret0, _ := ret[0].(*settings.GetValuesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSettingsClientRecorder) GetValues(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetValues", _s...)
}

func (_m *MockSettingsClient) RegisterConfigs(_param0 context.Context, _param1 *settings.RegisterConfigsRequest, _param2 ...grpc.CallOption) (*settings.RegisterConfigsResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "RegisterConfigs", _s...)
	ret0, _ := ret[0].(*settings.RegisterConfigsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSettingsClientRecorder) RegisterConfigs(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RegisterConfigs", _s...)
}

func (_m *MockSettingsClient) SetValue(_param0 context.Context, _param1 *settings.SetValueRequest, _param2 ...grpc.CallOption) (*settings.SetValueResponse, error) {
	_s := []interface{}{_param0, _param1}
	for _, _x := range _param2 {
		_s = append(_s, _x)
	}
	ret := _m.ctrl.Call(_m, "SetValue", _s...)
	ret0, _ := ret[0].(*settings.SetValueResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockSettingsClientRecorder) SetValue(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	_s := append([]interface{}{arg0, arg1}, arg2...)
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetValue", _s...)
}