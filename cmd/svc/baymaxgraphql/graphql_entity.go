package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
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

// ehrLink represents a link to an ehr for an entity
var ehrLinkType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "EHRLink",
	Description: "A link to an EHR for an entity",
	Fields: graphql.Fields{
		"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"url":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var entityType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Entity",
	Interfaces: []*graphql.Interface{
		nodeInterfaceType,
	},
	Fields: graphql.Fields{
		"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"isEditable": &graphql.Field{
			Type:              graphql.NewNonNull(graphql.Boolean),
			DeprecationReason: "Use allowEdit instead.",
		},
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
		"allowEdit": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"avatar": &graphql.Field{
			Type: imageType,
			Args: NewImageArguments(nil),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				svc := serviceFromParams(p)
				imgArgs := ParseImageArguments(p.Args)
				ent := p.Source.(*models.Entity)
				if ent.ImageMediaID == "" {
					return ent.Avatar, nil
				}
				// If no args were provided then default to the current avatar standard
				if imgArgs.Width == 0 && imgArgs.Height == 0 {
					imgArgs.Width = 108
					imgArgs.Height = 108
				}
				return &models.Image{
					URL:    media.ThumbnailURL(svc.mediaAPIDomain, ent.ImageMediaID, imgArgs.Height, imgArgs.Width, imgArgs.Crop),
					Width:  imgArgs.Width,
					Height: imgArgs.Height,
				}, nil
			},
		},
		"callableEndpoints":     &graphql.Field{Type: graphql.NewList(callEndpointType)},
		"hasAccount":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"hasPendingInvite":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"isInternal":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"lastModifiedTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"hasProfile":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"profile": &graphql.Field{
			Type: profileType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				ent := p.Source.(*models.Entity)
				ctx := p.Context
				ram := raccess.ResourceAccess(p)
				return lookupEntityProfile(ctx, ram, ent.ID)
			},
		},
		"ehrLinks": &graphql.Field{
			Type: graphql.NewList(ehrLinkType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				ent := p.Source.(*models.Entity)
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				res, err := ram.LookupEHRLinksForEntity(ctx, &directory.LookupEHRLinksForEntityRequest{
					EntityID: ent.ID,
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				transformedEHRLinks := make([]*models.EHRLink, len(res.Links))
				for i, ehrLink := range res.Links {
					transformedEHRLinks[i] = &models.EHRLink{
						Name: ehrLink.Name,
						URL:  ehrLink.URL,
					}
				}

				return transformedEHRLinks, nil
			},
		},
		"paymentMethods": &graphql.Field{
			Type:    graphql.NewList(graphql.NewNonNull(paymentMethodInterfaceType)),
			Resolve: apiaccess.Authenticated(resolveEntityPaymentMethods),
		},
	},
})

func resolveEntityPaymentMethods(p graphql.ResolveParams) (interface{}, error) {
	ent := p.Source.(*models.Entity)
	ctx := p.Context
	ram := raccess.ResourceAccess(p)

	resp, err := ram.PaymentMethods(ctx, &payments.PaymentMethodsRequest{
		EntityID: ent.ID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return transformPaymentMethodsToResponse(resp.PaymentMethods), nil
}

func lookupEntity(ctx context.Context, svc *service, ram raccess.ResourceAccessor, entityID string) (interface{}, error) {
	em, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, err
	}
	oc, err := transformContactsToResponse(em.Contacts)
	if err != nil {
		return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity contacts: %+v", err))
	}

	sh := devicectx.SpruceHeaders(ctx)
	switch em.Type {
	case directory.EntityType_ORGANIZATION:
		org := &models.Organization{
			ID:       em.ID,
			Name:     em.Info.DisplayName,
			Contacts: oc,
		}

		acc := gqlctx.Account(ctx)
		if acc != nil {
			e, err := entityInOrgForAccountID(ctx, ram, org.ID, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if e != nil {
				org.Entity, err = transformEntityToResponse(ctx, svc.staticURLPrefix, e, sh, gqlctx.Account(ctx))
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
			}
		}
		return org, nil
	case directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL, directory.EntityType_SYSTEM, directory.EntityType_PATIENT:
		e, err := transformEntityToResponse(ctx, svc.staticURLPrefix, em, sh, gqlctx.Account(ctx))
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return e, nil
	}
	return nil, errors.InternalError(ctx, fmt.Errorf("unknown entity type: %s", em.Type.String()))
}
