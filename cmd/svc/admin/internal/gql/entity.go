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

// entityField returns is a graphql field for Querying an Entity object
var entityField = &graphql.Field{
	Type:    entityType,
	Args:    entityArgumentsConfig,
	Resolve: entityResolve,
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
				"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"accountID":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"contacts":      &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactType))},
				"firstName":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"middleInitial": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"lastName":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"groupName":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"displayName":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"shortTitle":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"longTitle":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"gender":        &graphql.Field{Type: graphql.NewNonNull(genderEnumType)},
				"dob":           &graphql.Field{Type: dateType},
				"note":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"members":       &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(entityType))},
				"memberships":   &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(entityType))},
				"orgLink": &graphql.Field{Type: graphql.NewNonNull(graphql.String),
					Resolve:           resolvePracticeLink,
					DeprecationReason: "DEPRECATED due to practice links becoming plural per org. Use `practiceLinks`",
				},
				"practiceLinks":  &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(practiceLinkType)), Resolve: resolvePracticeLinks},
				"type":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"status":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"externalIDs":    &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
				"settings":       &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(settingType)), Resolve: resolveEntitySettings},
				"vendorAccounts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(vendorAccountType)), Resolve: resolveEntityVendorAccounts},
			},
		},
	)
}

func resolveEntitySettings(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	entity := p.Source.(*models.Entity)
	golog.ContextLogger(ctx).Debugf("Looking up entity settings for %s", entity.ID)
	return getNodeSettings(ctx, client.Settings(p), entity.ID)
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
		ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		golog.ContextLogger(ctx).Warningf("Error while fetching entity %s", err)
		return nil, errors.Trace(err)
	} else if len(resp.Entities) != 1 {
		return nil, errors.Errorf("Expected 1 result but got %v", resp.Entities)
	}
	return models.TransformEntityToModel(resp.Entities[0]), nil
}

// DEPRECATED
func resolvePracticeLink(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	entity := p.Source.(*models.Entity)
	golog.ContextLogger(ctx).Debugf("Looking up practice link for %s", entity.ID)
	practiceLinks, err := getPracticeLinksForEntity(ctx, client.Invite(p), client.Domains(p).InviteAPI, entity.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// For now on this old singular call just return the first once till all clients are ported and this is deprecated
	var link string
	if len(practiceLinks) != 0 {
		link = practiceLinks[0].URL
	}
	return link, nil
}

func resolvePracticeLinks(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	entity := p.Source.(*models.Entity)
	golog.ContextLogger(ctx).Debugf("Looking up practice links for %s", entity.ID)
	practiceLinks, err := getPracticeLinksForEntity(ctx, client.Invite(p), client.Domains(p).InviteAPI, entity.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return practiceLinks, nil
}
