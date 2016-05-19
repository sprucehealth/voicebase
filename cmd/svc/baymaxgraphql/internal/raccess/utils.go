package raccess

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
)

// EntityInOrgForAccountID returns the entity in the org specified from the account in the context.
func EntityInOrgForAccountID(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest, orgID string) (*directory.Entity, error) {
	// assert that the lookup entities request is for looking up an entity
	// via externalID
	if req.LookupKeyType != directory.LookupEntitiesRequest_EXTERNAL_ID {
		return nil, errors.Trace(fmt.Errorf("Expected lookup of type %s but got %s", directory.LookupEntitiesRequest_EXTERNAL_ID, req.LookupKeyType))
	}

	// Check our cached account entities first
	acc := gqlctx.Account(ctx)
	if acc != nil && acc.ID == req.GetExternalID() {
		ents := gqlctx.AccountEntities(ctx)
		if ents != nil {
			ent := ents.Get(orgID)
			if ent != nil {
				return ent, nil
			}
		}
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

	return nil, fmt.Errorf("(entity for account %s and org %s)", req.GetExternalID(), orgID)
}

// Entity returns a single expected entity for the directory request.
func Entity(ctx context.Context, ram ResourceAccessor, req *directory.LookupEntitiesRequest) (*directory.Entity, error) {

	if req.LookupKeyType != directory.LookupEntitiesRequest_ENTITY_ID && req.LookupKeyType != directory.LookupEntitiesRequest_EXTERNAL_ID {
		return nil, fmt.Errorf("Expected lookup of type %s but got %s", directory.LookupEntitiesRequest_ENTITY_ID, req.LookupKeyType)
	}

	entities, err := ram.Entities(ctx, req)
	if err != nil {
		return nil, err
	} else if len(entities) != 1 {
		id := req.GetEntityID()
		if id == "" {
			id = req.GetExternalID()
		}
		return nil, errors.Trace(fmt.Errorf("Expected 1 entity got %d for %s", len(entities), id))
	}

	return entities[0], nil
}
