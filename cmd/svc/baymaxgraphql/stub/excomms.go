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

// ProvisionEmailAddress provisions an email address for the requester.
func (sc *stubExCommsClient) ProvisionEmailAddress(ctx context.Context, in *excomms.ProvisionEmailAddressRequest, opts ...grpc.CallOption) (*excomms.ProvisionEmailAddressResponse, error) {
	return &excomms.ProvisionEmailAddressResponse{
		EmailAddress: harness.RandEmail(),
	}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (sc *stubExCommsClient) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	return nil, errNotSupportedInStub
}

// InitiatePhoneCall initiates a phone call as defined in the InitiatePhoneCallRequest.
func (sc *stubExCommsClient) InitiatePhoneCall(ctx context.Context, in *excomms.InitiatePhoneCallRequest, opts ...grpc.CallOption) (*excomms.InitiatePhoneCallResponse, error) {
	return nil, errNotSupportedInStub
}
