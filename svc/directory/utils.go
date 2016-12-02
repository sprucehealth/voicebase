package directory

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	ErrEntityNotFound = errors.New("entity not found")
)

// EntityIDs is a convenience method for retrieving ID's from a list
// Note: This could be made more gneeric using reflection but don't want the performance cost
func EntityIDs(es []*Entity) []string {
	ids := make([]string, len(es))
	for i, e := range es {
		ids[i] = e.ID
	}
	return ids
}

// SingleEntity returns a single entity for the given lookup request. If just 1 entity is not found an error is returned.
func SingleEntity(ctx context.Context, client DirectoryClient, req *LookupEntitiesRequest) (*Entity, error) {
	res, err := client.LookupEntities(ctx, req)
	if err != nil && grpc.Code(err) == codes.NotFound {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Entities) == 0 {
		return nil, ErrEntityNotFound
	} else if len(res.Entities) != 1 {
		var id string
		switch key := req.LookupKeyOneof.(type) {
		case *LookupEntitiesRequest_EntityID:
			id = key.EntityID
		case *LookupEntitiesRequest_ExternalID:
			id = key.ExternalID
		case *LookupEntitiesRequest_AccountID:
			id = key.AccountID
		}
		return nil, fmt.Errorf("expected single entity for %s but got %d", id, len(res.Entities))
	}
	return res.Entities[0], nil
}

// SingleEntityByContact returns a single entity for a given contact value. If just 1 entity not found error is returned.
func SingleEntityByContact(ctx context.Context, client DirectoryClient, req *LookupEntitiesByContactRequest) (*Entity, error) {
	res, err := client.LookupEntitiesByContact(ctx, req)
	if err != nil && grpc.Code(err) == codes.NotFound {
		return nil, ErrEntityNotFound
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Entities) == 0 {
		return nil, ErrEntityNotFound
	} else if len(res.Entities) != 1 {
		return nil, fmt.Errorf("expected single entity for %s but got %d", req.ContactValue, len(res.Entities))
	}

	return res.Entities[0], nil
}

// OnlyEntity returns the only entity in the list and any other state represents an error
func OnlyEntity(es []*Entity) (*Entity, error) {
	if len(es) != 1 {
		return nil, fmt.Errorf("Expected only 1 entity to be present in list, but found %v", es)
	}
	return es[0], nil
}

// FilterContacts returns a list of contacts that match the defined type
func FilterContacts(entity *Entity, contactType ContactType) []*Contact {
	contacts := make([]*Contact, 0, len(entity.Contacts))
	for _, contact := range entity.Contacts {
		if contact.ContactType == contactType {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}
