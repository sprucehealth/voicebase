package mock

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
)

var _ settings.SettingsClient = &Client{}

type Client struct {
	*mock.Expector
}

func New(t testing.TB) *Client {
	return &Client{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (s *Client) RegisterConfigs(ctx context.Context, in *settings.RegisterConfigsRequest, opts ...grpc.CallOption) (*settings.RegisterConfigsResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*settings.RegisterConfigsResponse), mock.SafeError(rets[1])
}

func (s *Client) GetConfigs(ctx context.Context, in *settings.GetConfigsRequest, opts ...grpc.CallOption) (*settings.GetConfigsResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*settings.GetConfigsResponse), mock.SafeError(rets[1])
}

func (s *Client) SetValue(ctx context.Context, in *settings.SetValueRequest, opts ...grpc.CallOption) (*settings.SetValueResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*settings.SetValueResponse), mock.SafeError(rets[1])
}

func (s *Client) GetValues(ctx context.Context, in *settings.GetValuesRequest, opts ...grpc.CallOption) (*settings.GetValuesResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*settings.GetValuesResponse), mock.SafeError(rets[1])
}
