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

// UpdateVendorAccount implements payments.PaymentsClient
func (c *Client) UpdateVendorAccount(ctx context.Context, in *payments.UpdateVendorAccountRequest, opts ...grpc.CallOption) (*payments.UpdateVendorAccountResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.UpdateVendorAccountResponse), mock.SafeError(rets[1])
}

// VendorAccounts implements payments.PaymentsClient
func (c *Client) VendorAccounts(ctx context.Context, in *payments.VendorAccountsRequest, opts ...grpc.CallOption) (*payments.VendorAccountsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.VendorAccountsResponse), mock.SafeError(rets[1])
}

// CreatePaymentMethod implements payments.PaymentsClient
func (c *Client) CreatePaymentMethod(ctx context.Context, in *payments.CreatePaymentMethodRequest, opts ...grpc.CallOption) (*payments.CreatePaymentMethodResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.CreatePaymentMethodResponse), mock.SafeError(rets[1])
}

// DeletePaymentMethod implements payments.PaymentsClient
func (c *Client) DeletePaymentMethod(ctx context.Context, in *payments.DeletePaymentMethodRequest, opts ...grpc.CallOption) (*payments.DeletePaymentMethodResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.DeletePaymentMethodResponse), mock.SafeError(rets[1])
}

// PaymentMethods implements payments.PaymentsClient
func (c *Client) PaymentMethods(ctx context.Context, in *payments.PaymentMethodsRequest, opts ...grpc.CallOption) (*payments.PaymentMethodsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*payments.PaymentMethodsResponse), mock.SafeError(rets[1])
}