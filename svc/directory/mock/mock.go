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

// CreateContacts implements directory.DirectoryClient
func (c *Client) CreateContacts(ctx context.Context, in *directory.CreateContactsRequest, opts ...grpc.CallOption) (*directory.CreateContactsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateContactsResponse), mock.SafeError(rets[1])
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

// DeleteContacts implements directory.DirectoryClient
func (c *Client) DeleteContacts(ctx context.Context, in *directory.DeleteContactsRequest, opts ...grpc.CallOption) (*directory.DeleteContactsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.DeleteContactsResponse), mock.SafeError(rets[1])
}

// DeleteEntity implements directory.DirectoryClient
func (c *Client) DeleteEntity(ctx context.Context, in *directory.DeleteEntityRequest, opts ...grpc.CallOption) (*directory.DeleteEntityResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.DeleteEntityResponse), mock.SafeError(rets[1])
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

// CreateExternalIDs returns the external ids that map to a set of entity ids
func (c *Client) CreateExternalIDs(ctx context.Context, in *directory.CreateExternalIDsRequest, opts ...grpc.CallOption) (*directory.CreateExternalIDsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateExternalIDsResponse), mock.SafeError(rets[1])
}

// LookupEntityDomain returns the domain for the provided entity info
func (c *Client) LookupEntityDomain(ctx context.Context, in *directory.LookupEntityDomainRequest, opts ...grpc.CallOption) (*directory.LookupEntityDomainResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.LookupEntityDomainResponse), mock.SafeError(rets[1])
}

// UpdateContacts returns the external ids that map to a set of entity ids
func (c *Client) UpdateContacts(ctx context.Context, in *directory.UpdateContactsRequest, opts ...grpc.CallOption) (*directory.UpdateContactsResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.UpdateContactsResponse), mock.SafeError(rets[1])
}

// CreateEntityDomain created a domain for the provided entity info
func (c *Client) CreateEntityDomain(ctx context.Context, in *directory.CreateEntityDomainRequest, opts ...grpc.CallOption) (*directory.CreateEntityDomainResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.CreateEntityDomainResponse), mock.SafeError(rets[1])
}

func (c *Client) UpdateEntityDomain(ctx context.Context, in *directory.UpdateEntityDomainRequest, opts ...grpc.CallOption) (*directory.UpdateEntityDomainResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.UpdateEntityDomainResponse), mock.SafeError(rets[1])
}

// UpdateEntity returns the external ids that map to a set of entity ids
func (c *Client) UpdateEntity(ctx context.Context, in *directory.UpdateEntityRequest, opts ...grpc.CallOption) (*directory.UpdateEntityResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.UpdateEntityResponse), mock.SafeError(rets[1])
}

// SerializedEntityContact returns the platform specific serialized contact for an entity
func (c *Client) SerializedEntityContact(ctx context.Context, in *directory.SerializedEntityContactRequest, opts ...grpc.CallOption) (*directory.SerializedEntityContactResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*directory.SerializedEntityContactResponse), mock.SafeError(rets[1])
}
