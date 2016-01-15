package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ directory.DirectoryClient = &Client{}

// Client is a mock for the directory service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t *testing.T) *Client {
	return &Client{&mock.Expector{T: t}}
}

// LookupEntities implements directory.DirectoryClient
func (c *Client) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.LookupEntitiesResponse), mock.SafeError(rets[1])
}

// CreateEntity implements directory.DirectoryClient
func (c *Client) CreateEntity(ctx context.Context, in *directory.CreateEntityRequest, opts ...grpc.CallOption) (*directory.CreateEntityResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateEntityResponse), mock.SafeError(rets[1])
}

// CreateMembership implements directory.DirectoryClient
func (c *Client) CreateMembership(ctx context.Context, in *directory.CreateMembershipRequest, opts ...grpc.CallOption) (*directory.CreateMembershipResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateMembershipResponse), mock.SafeError(rets[1])
}

// LookupEntitiesByContact implements directory.DirectoryClient
func (c *Client) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.LookupEntitiesByContactResponse), mock.SafeError(rets[1])
}

// CreateContact implements directory.DirectoryClient
func (c *Client) CreateContact(ctx context.Context, in *directory.CreateContactRequest, opts ...grpc.CallOption) (*directory.CreateContactResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateContactResponse), mock.SafeError(rets[1])
}

// ExternalIDs returns the external ids that map to a set of entity ids
func (c *Client) ExternalIDs(ctx context.Context, in *directory.ExternalIDsRequest, opts ...grpc.CallOption) (*directory.ExternalIDsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.ExternalIDsResponse), mock.SafeError(rets[1])
}
