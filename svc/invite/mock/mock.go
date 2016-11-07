package mock

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/invite"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ invite.InviteClient = &Client{}

// Client is a mock for the invite service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

// AttributionData returns the attribution data for a device
func (c *Client) AttributionData(ctx context.Context, in *invite.AttributionDataRequest, opts ...grpc.CallOption) (*invite.AttributionDataResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.AttributionDataResponse), mock.SafeError(ret[1])
}

// CreateOrganizationInvite sends invites to people to join an organization
func (c *Client) CreateOrganizationInvite(ctx context.Context, in *invite.CreateOrganizationInviteRequest, opts ...grpc.CallOption) (*invite.CreateOrganizationInviteResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.CreateOrganizationInviteResponse), mock.SafeError(ret[1])
}

// InviteColleagues sends invites to people to join an organization
func (c *Client) InviteColleagues(ctx context.Context, in *invite.InviteColleaguesRequest, opts ...grpc.CallOption) (*invite.InviteColleaguesResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.InviteColleaguesResponse), mock.SafeError(ret[1])
}

// InvitePatients sends invites to people to join an organization
func (c *Client) InvitePatients(ctx context.Context, in *invite.InvitePatientsRequest, opts ...grpc.CallOption) (*invite.InvitePatientsResponse, error) {
	ret := c.Expector.Record(in)
	if len(ret) == 0 {
		return nil, nil
	}
	return ret[0].(*invite.InvitePatientsResponse), mock.SafeError(ret[1])
}

// LookupInvite returns information about an invite by token
func (c *Client) LookupInvite(ctx context.Context, in *invite.LookupInviteRequest, opts ...grpc.CallOption) (*invite.LookupInviteResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.LookupInviteResponse), mock.SafeError(ret[1])
}

// LookupInvites is a mock
func (c *Client) LookupInvites(ctx context.Context, in *invite.LookupInvitesRequest, opts ...grpc.CallOption) (*invite.LookupInvitesResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.LookupInvitesResponse), mock.SafeError(ret[1])
}

// LookupOrganizationInvites is a mock
func (c *Client) LookupOrganizationInvites(ctx context.Context, in *invite.LookupOrganizationInvitesRequest, opts ...grpc.CallOption) (*invite.LookupOrganizationInvitesResponse, error) {
	ret := c.Expector.Record(in)
	return ret[0].(*invite.LookupOrganizationInvitesResponse), mock.SafeError(ret[1])
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

// DeleteInvite is a mock
func (c *Client) DeleteInvite(ctx context.Context, in *invite.DeleteInviteRequest, opts ...grpc.CallOption) (*invite.DeleteInviteResponse, error) {
	ret := c.Expector.Record(in)
	if len(ret) == 0 {
		return nil, nil
	}
	return ret[0].(*invite.DeleteInviteResponse), mock.SafeError(ret[1])
}
