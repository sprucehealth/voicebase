package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
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
			"type":        &graphql.Field{Type: graphql.NewNonNull(contactEnumType)},
			"value":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"provisioned": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

var entityType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Entity",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":       &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name":     &graphql.Field{Type: graphql.String},
			"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
			// TODO: avatar(width: Int = 120, height: Int = 120, crop: Boolean = true): Image
		},
	},
)

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
		return nil, internalError(err)
	}
	for _, em := range res.Entities {
		oc, err := transformContactsToResponse(em.Contacts)
		if err != nil {
			return nil, internalError(fmt.Errorf("failed to transform entity contacts: %+v", err))
		}
		switch em.Type {
		case directory.EntityType_ORGANIZATION:
			org := &organization{
				ID:       em.ID,
				Name:     em.Name,
				Contacts: oc,
			}

			acc := accountFromContext(ctx)
			if acc != nil {
				e, err := svc.entityForAccountID(ctx, org.ID, acc.ID)
				if err != nil {
					return nil, internalError(err)
				}
				if e != nil {
					org.Entity, err = transformEntityToResponse(e)
					if err != nil {
						return nil, internalError(err)
					}
				}
			}
			return org, nil
		case directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL:
			return &entity{
				ID:       em.ID,
				Name:     em.Name,
				Contacts: oc,
			}, nil
		default:
			return nil, internalError(fmt.Errorf("unknown entity type: %s", em.Type.String()))
		}
	}
	return nil, errors.New("not found")
}
