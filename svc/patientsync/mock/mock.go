package mock

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/patientsync"
)

type Client struct {
	*mock.Expector
}

func New(t *testing.T) *Client {
	return &Client{
		Expector: &mock.Expector{
			T: t,
		},
	}
}

func (s *Client) ConfigureSync(ctx context.Context, in *patientsync.ConfigureSyncRequest, opts ...grpc.CallOption) (*patientsync.ConfigureSyncResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*patientsync.ConfigureSyncResponse), mock.SafeError(rets[1])
}

func (s *Client) InitiateSync(ctx context.Context, in *patientsync.InitiateSyncRequest, opts ...grpc.CallOption) (*patientsync.InitiateSyncResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*patientsync.InitiateSyncResponse), mock.SafeError(rets[1])
}
func (s *Client) LookupSyncConfiguration(ctx context.Context, in *patientsync.LookupSyncConfigurationRequest, opts ...grpc.CallOption) (*patientsync.LookupSyncConfigurationResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*patientsync.LookupSyncConfigurationResponse), mock.SafeError(rets[1])
}

func (s *Client) UpdateSyncConfiguration(ctx context.Context, in *patientsync.UpdateSyncConfigurationRequest, opts ...grpc.CallOption) (*patientsync.UpdateSyncConfigurationResponse, error) {
	rets := s.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*patientsync.UpdateSyncConfigurationResponse), mock.SafeError(rets[1])
}
