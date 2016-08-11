package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

// entityArgumentsConfig represents the config for arguments referencing an entity
var entityArgumentsConfig = graphql.FieldConfigArgument{
	"id": &graphql.ArgumentConfig{Type: graphql.String},
}

// entityArguments represents arguments for referencing an entity
type entityArguments struct {
	ID string `json:"id"`
}

// parseEntityArguments parses the entity arguments out of requests params
func parseEntityArguments(args map[string]interface{}) *entityArguments {
	entArgs := &entityArguments{}
	if args != nil {
		if iid, ok := args["id"]; ok {
			if id, ok := iid.(string); ok {
				entArgs.ID = id
			}
		}
	}
	return entArgs
}

// newEntityField returns a graphql field for Querying an Entity object
func newEntityField() *graphql.Field {
	return &graphql.Field{
		Type:    newEntityType(),
		Args:    entityArgumentsConfig,
		Resolve: entityResolve,
	}
}

func entityResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	args := parseEntityArguments(p.Args)
	golog.ContextLogger(ctx).Debugf("Resolving Entity with args %+v", args)
	if args.ID == "" {
		return nil, nil
	}
	return getEntity(ctx, client.Directory(p), args.ID)
}

// special case the way we handle entity types since they are recursive
var entityType = &graphql.Object{}

func init() {
	*entityType = *graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Entity",
			Fields: graphql.Fields{
				"id":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"type":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"firstName":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"middleInitial":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"lastName":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"groupName":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"displayName":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"shortTitle":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"longTitle":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"gender":         &graphql.Field{Type: graphql.NewNonNull(genderEnumType)},
				"dob":            &graphql.Field{Type: newDateType()},
				"note":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"members":        &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(newEntityType()))},
				"memberships":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(newEntityType()))},
				"contacts":       &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(newContactType()))},
				"externalIDs":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
				"settings":       &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(newSettingType())), Resolve: resolveEntitySettings},
				"vendorAccounts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(newVendorAccountType())), Resolve: resolveEntityVendorAccounts},
			},
		},
	)
}

// newEntityType returns an instance of the Entity graphql type. This is just sugar to follow the pattern
func newEntityType() *graphql.Object {
	return entityType
}

func resolveEntitySettings(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	entity := p.Source.(*models.Entity)
	golog.ContextLogger(ctx).Debugf("Looking up entity settings for %s", entity.ID)
	return getEntitySettings(ctx, client.Settings(p), entity.ID)
}

func resolveEntityVendorAccounts(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	entity := p.Source.(*models.Entity)
	golog.ContextLogger(ctx).Debugf("Looking up entity vendor accounts for %s", entity.ID)
	return getEntityVendorAccounts(ctx, client.Payments(p), entity.ID)
}

func getEntity(ctx context.Context, dirCli directory.DirectoryClient, id string) (*models.Entity, error) {
	resp, err := dirCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: id,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
	})
	if err != nil {
		golog.ContextLogger(ctx).Warningf("Error while fetching entity %s", err)
		return nil, errors.Trace(err)
	} else if len(resp.Entities) != 1 {
		return nil, errors.Errorf("Expected 1 result but got %v", resp.Entities)
	}
	return models.TransformEntityToModel(resp.Entities[0]), nil
}
