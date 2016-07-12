package stub

import (
	"context"

	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
)

type excommsClient struct {
}

func NewClient() excomms.ExCommsClient {
	return &excommsClient{}
}

// SearchAvailablephoneNumbers returns a list of available phone numbers based on the search criteria.
func (e *excommsClient) SearchAvailablePhoneNumbers(ctx context.Context, in *excomms.SearchAvailablePhoneNumbersRequest, opts ...grpc.CallOption) (*excomms.SearchAvailablePhoneNumbersResponse, error) {
	return &excomms.SearchAvailablePhoneNumbersResponse{}, nil
}

// ProvisionPhoneNumber provisions the phone number provided for the requester.
func (e *excommsClient) ProvisionPhoneNumber(ctx context.Context, in *excomms.ProvisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.ProvisionPhoneNumberResponse, error) {
	return &excomms.ProvisionPhoneNumberResponse{}, nil
}

func (e *excommsClient) DeprovisionPhoneNumber(ctx context.Context, in *excomms.DeprovisionPhoneNumberRequest, opts ...grpc.CallOption) (*excomms.DeprovisionPhoneNumberResponse, error) {
	return &excomms.DeprovisionPhoneNumberResponse{}, nil
}

// ProvisionEmailAddress provisions an email address for the requester.
func (e *excommsClient) ProvisionEmailAddress(ctx context.Context, in *excomms.ProvisionEmailAddressRequest, opts ...grpc.CallOption) (*excomms.ProvisionEmailAddressResponse, error) {
	return &excomms.ProvisionEmailAddressResponse{}, nil
}

// SendMessage sends the message over an external channel as specified in the SendMessageRequest.
func (e *excommsClient) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	return &excomms.SendMessageResponse{}, nil
}

// InitiatePhoneCall initiates a phone call as defined in the InitiatePhoneCallRequest.
func (e *excommsClient) InitiatePhoneCall(ctx context.Context, in *excomms.InitiatePhoneCallRequest, opts ...grpc.CallOption) (*excomms.InitiatePhoneCallResponse, error) {
	return &excomms.InitiatePhoneCallResponse{}, nil
}

func (e *excommsClient) DeprovisionEmail(ctx context.Context, in *excomms.DeprovisionEmailRequest, opts ...grpc.CallOption) (*excomms.DeprovisionEmailResponse, error) {
	return &excomms.DeprovisionEmailResponse{}, nil
}

func (e *excommsClient) InitiateIPCall(ctx context.Context, in *excomms.InitiateIPCallRequest, opts ...grpc.CallOption) (*excomms.InitiateIPCallResponse, error) {
	return &excomms.InitiateIPCallResponse{}, nil
}

func (e *excommsClient) IPCall(ctx context.Context, in *excomms.IPCallRequest, opts ...grpc.CallOption) (*excomms.IPCallResponse, error) {
	return &excomms.IPCallResponse{}, nil
}

func (e *excommsClient) PendingIPCalls(ctx context.Context, in *excomms.PendingIPCallsRequest, opts ...grpc.CallOption) (*excomms.PendingIPCallsResponse, error) {
	return &excomms.PendingIPCallsResponse{}, nil
}

func (e *excommsClient) UpdateIPCall(ctx context.Context, in *excomms.UpdateIPCallRequest, opts ...grpc.CallOption) (*excomms.UpdateIPCallResponse, error) {
	return &excomms.UpdateIPCallResponse{}, nil
}
