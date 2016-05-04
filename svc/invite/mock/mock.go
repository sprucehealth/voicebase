package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/invite"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ invite.InviteClient = &Client{}

// Client is a mock for the invite service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t *testing.T) *Client {
	return &Client{&mock.Expector{T: t}}
}

// AttributionData returns the attribution data for a device
func (c *Client) AttributionData(ctx context.Context, in *invite.AttributionDataRequest, opts ...grpc.CallOption) (*invite.AttributionDataResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.AttributionDataResponse), mock.SafeError(ret[1])
}

// InviteColleagues sends invites to people to join an organization
func (c *Client) InviteColleagues(ctx context.Context, in *invite.InviteColleaguesRequest, opts ...grpc.CallOption) (*invite.InviteColleaguesResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.InviteColleaguesResponse), mock.SafeError(ret[1])
}

// InvitePatients sends invites to people to join an organization
func (c *Client) InvitePatients(ctx context.Context, in *invite.InvitePatientsRequest, opts ...grpc.CallOption) (*invite.InvitePatientsResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.InvitePatientsResponse), mock.SafeError(ret[1])
}

// LookupInvite returns information about an invite by token
func (c *Client) LookupInvite(ctx context.Context, in *invite.LookupInviteRequest, opts ...grpc.CallOption) (*invite.LookupInviteResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.LookupInviteResponse), mock.SafeError(ret[1])
}

// SetAttributionData associate attribution data with a device
func (c *Client) SetAttributionData(ctx context.Context, in *invite.SetAttributionDataRequest, opts ...grpc.CallOption) (*invite.SetAttributionDataResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.SetAttributionDataResponse), mock.SafeError(ret[1])
}

// MarkInviteConsumed deleted an invite
func (c *Client) MarkInviteConsumed(ctx context.Context, in *invite.MarkInviteConsumedRequest, opts ...grpc.CallOption) (*invite.MarkInviteConsumedResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.MarkInviteConsumedResponse), mock.SafeError(ret[1])
}
