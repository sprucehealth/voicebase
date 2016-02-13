package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		"note":          &graphql.Field{Type: graphql.String},
		"initials": &graphql.Field{
			Type: graphql.NewNonNull(graphql.String),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*entity)
				var first, last string
				if entity.FirstName != "" {
					first = entity.FirstName[:1]
				}
				if entity.LastName != "" {
					last = entity.LastName[:1]
				}
				return strings.ToUpper(first + last), nil
			},
		},
		"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
		"serializedContact": &graphql.Field{
			Type: graphql.String,
			Args: graphql.FieldConfigArgument{
				"platform": &graphql.ArgumentConfig{Type: graphql.NewNonNull(platformEnumType)},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*entity)
				svc := serviceFromParams(p)
				ctx := p.Context
				acc := accountFromContext(ctx)
				if acc == nil {
					return nil, errNotAuthenticated(ctx)
				}

				platform, _ := p.Args["platform"].(string)
				pPlatform, ok := directory.Platform_value[platform]
				if !ok {
					return nil, fmt.Errorf("Unknown platform type %s", platform)
				}
				dPlatform := directory.Platform(pPlatform)

				sc, err := lookupSerializedEntityContact(ctx, svc, entity.ID, dPlatform)
				if err != nil {
					return nil, internalError(ctx, err)
				}

				return sc, nil
			},
		},
		"avatar": &graphql.Field{
			Type: imageType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				entity := p.Source.(*entity)
				// TODO: should have arugments for width, height, crop, etc..
				return entity.avatar, nil
			},
		},
		"isInternal": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

func lookupEntity(ctx context.Context, svc *service, id string) (interface{}, error) {
	res, err := svc.directory.LookupEntities(ctx,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: id,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, errors.New("not found")
		}
		return nil, internalError(ctx, err)
	}
	for _, em := range res.Entities {
		oc, err := transformContactsToResponse(em.Contacts)
		if err != nil {
			return nil, internalError(ctx, fmt.Errorf("failed to transform entity contacts: %+v", err))
		}
		switch em.Type {
		case directory.EntityType_ORGANIZATION:
			org := &organization{
				ID:       em.ID,
				Name:     em.Info.DisplayName,
				Contacts: oc,
			}

			acc := accountFromContext(ctx)
			if acc != nil {
				e, err := svc.entityForAccountID(ctx, org.ID, acc.ID)
				if err != nil {
					return nil, internalError(ctx, err)
				}
				if e != nil {
					org.Entity, err = transformEntityToResponse(e)
					if err != nil {
						return nil, internalError(ctx, err)
					}
				}
			}
			return org, nil
		case directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL, directory.EntityType_SYSTEM:
			e, err := transformEntityToResponse(em)
			if err != nil {
				return nil, internalError(ctx, err)
			}
			return e, nil
		default:
			return nil, internalError(ctx, fmt.Errorf("unknown entity type: %s", em.Type.String()))
		}
	}
	return nil, errors.New("not found")
}
