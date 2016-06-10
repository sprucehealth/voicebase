package stub

import (
	"errors"

	"github.com/sprucehealth/backend/cmd/svc/blackbox/harness"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewStubExcommsClient returns an initialized instance of stubExCommsClient
func NewStubExcommsClient() excomms.ExCommsClient {
	return &stubExCommsClient{}
}

var errNotSupportedInStub = errors.New("Not supported in stub excomms client")

type stubExCommsClient struct{}

// SearchAvailablephoneNumbers returns a list of available phone numbers based on the search criteria.
func (sc *stubExCommsClient) SearchAvailablePhoneNumbers(ctx context.Context, in *excomms.SearchAvailablePhoneNumbersRequest, opts ...grpc.CallOption) (*excomms.SearchAvailablePhoneNumbersResponse, error) {
	return nil, errNotSupportedInStub
}

// ProvisionPhoneNumber provisions the phone number provided for the requester.
func (sc *stubExCommsClient) ProvisionPhoneNumber(ctx context.Context, in *excomms.ProvisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.ProvisionPhoneNumberResponse, error) {
	return &excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: harness.RandPhoneNumber(),
	}, nil
}

func (sc *stubExCommsClient) DeprovisionPhoneNumber(ctx context.Context, in *excomms.DeprovisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.DeprovisionPhoneNumberResponse, error) {
	return &excomms.DeprovisionPhoneNumberResponse{}, nil
}

// ProvisionEmailAddress provisions an email address for the requester.
func (sc *stubExCommsClient) ProvisionEmailAddress(ctx context.Context, in *excomms.ProvisionEmailAddressRequest, opts ...grpc.CallOption) (*excomms.ProvisionEmailAddressResponse, error) {
	return &excomms.ProvisionEmailAddressResponse{
		EmailAddress: harness.RandEmail(),
	}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (sc *stubExCommsClient) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	return &excomms.SendMessageResponse{}, nil
}

// InitiatePhoneCall initiates a phone call as defined in the InitiatePhoneCallRequest.
func (sc *stubExCommsClient) InitiatePhoneCall(ctx context.Context, in *excomms.InitiatePhoneCallRequest, opts ...grpc.CallOption) (*excomms.InitiatePhoneCallResponse, error) {
	return nil, errNotSupportedInStub
}

func (sc *stubExCommsClient) DeprovisionEmail(ctx context.Context, in *excomms.DeprovisionEmailRequest, opts ...grpc.CallOption) (*excomms.DeprovisionEmailResponse, error) {
	return &excomms.DeprovisionEmailResponse{}, nil
}

func (sc *stubExCommsClient) InitiateIPCall(ctx context.Context, in *excomms.InitiateIPCallRequest, opts ...grpc.CallOption) (*excomms.InitiateIPCallResponse, error) {
	return &excomms.InitiateIPCallResponse{}, nil
}

func (sc *stubExCommsClient) PendingIPCalls(ctx context.Context, in *excomms.PendingIPCallsRequest, opts ...grpc.CallOption) (*excomms.PendingIPCallsResponse, error) {
	return &excomms.PendingIPCallsResponse{}, nil
}

func (sc *stubExCommsClient) UpdateIPCall(ctx context.Context, in *excomms.UpdateIPCallRequest, opts ...grpc.CallOption) (*excomms.UpdateIPCallResponse, error) {
	return &excomms.UpdateIPCallResponse{}, nil
}
