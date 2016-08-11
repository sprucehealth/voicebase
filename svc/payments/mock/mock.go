package mock

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/payments"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ payments.PaymentsClient = &Client{}

// Client is a mock for the directory service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

// ConnectVendorAccount implements payments.PaymentsClient
func (c *Client) ConnectVendorAccount(ctx context.Context, in *payments.ConnectVendorAccountRequest, opts ...grpc.CallOption) (*payments.ConnectVendorAccountResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.ConnectVendorAccountResponse), mock.SafeError(rets[1])
}

// DisconnectVendorAccount implements payments.PaymentsClient
func (c *Client) DisconnectVendorAccount(ctx context.Context, in *payments.DisconnectVendorAccountRequest, opts ...grpc.CallOption) (*payments.DisconnectVendorAccountResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.DisconnectVendorAccountResponse), mock.SafeError(rets[1])
}

// VendorAccounts implements payments.PaymentsClient
func (c *Client) VendorAccounts(ctx context.Context, in *payments.VendorAccountsRequest, opts ...grpc.CallOption) (*payments.VendorAccountsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.VendorAccountsResponse), mock.SafeError(rets[1])
}
