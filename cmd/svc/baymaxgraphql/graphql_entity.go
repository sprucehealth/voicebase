package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var contactEnumType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ContactType",
		Description: "Type of contact value",
		Values: graphql.EnumValueConfigMap{
			"APP": &graphql.EnumValueConfig{
				Value:       "APP",
				Description: "Application or web",
			},
			"PHONE": &graphql.EnumValueConfig{
				Value:       "PHONE",
				Description: "Phone",
			},
			"EMAIL": &graphql.EnumValueConfig{
				Value:       "EMAIL",
				Description: "Email",
			},
		},
	},
)

var contactInfoType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ContactInfo",
		Fields: graphql.Fields{
			"id":           &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":         &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
			"value":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"displayValue": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"provisioned":  &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"label":        &graphql.Field{Type: graphql.String},
		},
	},
)

// dateType represents a date
var dateType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Date",
	Fields: graphql.Fields{
		"month": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"day":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"year":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})

var entityType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Entity",
	Interfaces: []*graphql.Interface{
		nodeInterfaceType,
	},
	Fields: graphql.Fields{
		"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"isEditable":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"firstName":     &graphql.Field{Type: graphql.String},
		"middleInitial": &graphql.Field{Type: graphql.String},
		"lastName":      &graphql.Field{Type: graphql.String},
		"groupName":     &graphql.Field{Type: graphql.String},
		"displayName":   &graphql.Field{Type: graphql.String},
		"longTitle":     &graphql.Field{Type: graphql.String},
		"shortTitle":    &graphql.Field{Type: graphql.String},
		"gender":        &graphql.Field{Type: graphql.NewNonNull(genderEnumType)},
		"dob":           &graphql.Field{Type: dateType},
		"note":          &graphql.Field{Type: graphql.String},
		"initials": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*models.Entity)
				return initialsForEntity(entity), nil
			},
		},
		"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
		"serializedContact": &graphql.Field{
			Type: graphql.String,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnumType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*models.Entity)
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)
				if acc == nil {
					return nil, errors.ErrNotAuthenticated(ctx)
				}

				platform, _ := p.Args["platform"].(string)
				pPlatform, ok := directory.Platform_value[platform]
				if !ok {
					return nil, fmt.Errorf("Unknown platform type %s", platform)
				}
				dPlatform := directory.Platform(pPlatform)

				sc, err := lookupSerializedEntityContact(ctx, ram, entity.ID, dPlatform)

				if err != nil {
					if errors.Type(err) == errors.ErrTypeNotFound {
						return nil, nil
					}
					return nil, errors.InternalError(ctx, err)
				}

				return sc, nil
			},
		},
		"avatar": &graphql.Field{
			Type: imageType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*models.Entity)
				// TODO: should have arugments for width, height, crop, etc..
				return entity.Avatar, nil
			},
		},
		"isInternal":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"lastModifiedTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"hasAccount":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"allowEdit":             &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

func lookupEntity(ctx context.Context, svc *service, ram raccess.ResourceAccessor, entityID string) (interface{}, error) {
	em, err := ram.Entity(ctx, entityID, []directory.EntityInformation{directory.EntityInformation_CONTACTS}, 0)
	if err != nil {
		return nil, err
	}
	oc, err := transformContactsToResponse(em.Contacts)
	if err != nil {
		return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity contacts: %+v", err))
	}

	sh := gqlctx.SpruceHeaders(ctx)
	switch em.Type {
	case directory.EntityType_ORGANIZATION:
		org := &models.Organization{
			ID:       em.ID,
			Name:     em.Info.DisplayName,
			Contacts: oc,
		}

		acc := gqlctx.Account(ctx)
		if acc != nil {
			e, err := ram.EntityForAccountID(ctx, org.ID, acc.ID)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if e != nil {
				org.Entity, err = transformEntityToResponse(svc.staticURLPrefix, e, sh, gqlctx.Account(ctx))
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}
		}
		return org, nil
	case directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL, directory.EntityType_SYSTEM:
		e, err := transformEntityToResponse(svc.staticURLPrefix, em, sh, gqlctx.Account(ctx))
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return e, nil
	}
	return nil, errors.InternalError(ctx, fmt.Errorf("unknown entity type: %s", em.Type.String()))
}
