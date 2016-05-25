package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ excomms.ExCommsClient = &Client{}

type Client struct {
	*mock.Expector
}

func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

func (c *Client) SearchAvailablePhoneNumbers(ctx context.Context, in *excomms.SearchAvailablePhoneNumbersRequest, opts ...grpc.CallOption) (*excomms.SearchAvailablePhoneNumbersResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.SearchAvailablePhoneNumbersResponse), mock.SafeError(rets[1])
}

func (c *Client) ProvisionPhoneNumber(ctx context.Context, in *excomms.ProvisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.ProvisionPhoneNumberResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.ProvisionPhoneNumberResponse), mock.SafeError(rets[1])
}

func (c *Client) DeprovisionPhoneNumber(ctx context.Context, in *excomms.DeprovisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.DeprovisionPhoneNumberResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*excomms.DeprovisionPhoneNumberResponse), mock.SafeError(rets[1])
}

func (c *Client) ProvisionEmailAddress(ctx context.Context, in *excomms.ProvisionEmailAddressRequest, opts ...grpc.CallOption) (*excomms.ProvisionEmailAddressResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*excomms.ProvisionEmailAddressResponse), mock.SafeError(rets[1])
}

func (c *Client) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*excomms.SendMessageResponse), mock.SafeError(rets[1])
}

func (c *Client) InitiatePhoneCall(ctx context.Context, in *excomms.InitiatePhoneCallRequest, opts ...grpc.CallOption) (*excomms.InitiatePhoneCallResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*excomms.InitiatePhoneCallResponse), mock.SafeError(rets[1])
}

func (c *Client) DeprovisionEmail(ctx context.Context, in *excomms.DeprovisionEmailRequest, opts ...grpc.CallOption) (*excomms.DeprovisionEmailResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*excomms.DeprovisionEmailResponse), mock.SafeError(rets[1])
}
