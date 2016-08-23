package raccess

import (
	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
)

// EntityInOrgForAccountID returns the entity in the org specified from the account in the context.
func EntityInOrgForAccountID(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest, orgID string) (*directory.Entity, error) {
	// assert that the lookup entities request is for looking up an entity
	// via externalID
	if req.LookupKeyType != directory.LookupEntitiesRequest_EXTERNAL_ID {
		return nil, errors.Errorf("Expected lookup of type %s but got %s", directory.LookupEntitiesRequest_EXTERNAL_ID, req.LookupKeyType)
	}

	entities, err := ram.Entities(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, entity := range entities {
		for _, member := range entity.GetMemberships() {
			if member.Type == directory.EntityType_ORGANIZATION && member.ID == orgID {
				return entity, nil
			}
		}
	}

	return nil, errors.Errorf("Did not find entity for account %s and org %s", req.GetExternalID(), orgID)
}

// EntityForAccountID returns the entity for an account.
// TODO: this assumes there's only one active entity per account which is currently always the case
func EntityForAccountID(ctx context.Context, ram ResourceAccessor, accountID string) (*directory.Entity, error) {
	ent, err := Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	return ent, errors.Trace(err)
}

// Entity returns a single expected entity for the directory request.
func Entity(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest) (*directory.Entity, error) {
	return entity(ctx, ram, req)
}

// UnauthorizedEntity returns a single expected entity for the directory request.
func UnauthorizedEntity(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest) (*directory.Entity, error) {
	return entity(ctx, ram, req, EntityQueryOptionUnathorized)
}

// Entity returns a single expected entity for the directory request.
func entity(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest, opts ...EntityQueryOption) (*directory.Entity, error) {
	if req.LookupKeyType != directory.LookupEntitiesRequest_ENTITY_ID && req.LookupKeyType != directory.LookupEntitiesRequest_EXTERNAL_ID {
		return nil, errors.Errorf("Expected lookup of type %s but got %s", directory.LookupEntitiesRequest_ENTITY_ID, req.LookupKeyType)
	}

	entities, err := ram.Entities(ctx, req, opts...)
	if err != nil {
		return nil, err
	} else if len(entities) == 0 {
		return nil, ErrNotFound
	} else if len(entities) != 1 {
		id := req.GetEntityID()
		if id == "" {
			id = req.GetExternalID()
		}
		return nil, errors.Errorf("Expected 1 entity got %d for %s", len(entities), id)
	}

	return entities[0], nil
}
