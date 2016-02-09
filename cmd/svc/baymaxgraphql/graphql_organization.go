package main

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var organizationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Organization",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"entity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*organization)
					if org.Entity != nil {
						return org.Entity, nil
					}

					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated
					}

					e, err := svc.entityForAccountID(ctx, org.ID, acc.ID)
					if err != nil {
						return nil, internalError(err)
					}
					if e == nil {
						return nil, errors.New("entity not found for organization")
					}
					rE, err := transformEntityToResponse(e)
					if err != nil {
						return nil, internalError(err)
					}
					return rE, nil
				},
			},
			"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
			"entities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*organization)
					if org.Entity == nil || org.Entity.ID == "" {
						return nil, errors.New("no entity for organization")
					}
					svc := serviceFromParams(p)
					ctx := p.Context

					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: org.ID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 0,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_MEMBERS,
									// TODO: don't always need contacts
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if grpc.Code(err) == codes.NotFound {
						return nil, errors.New("not found")
					} else if err != nil {
						return nil, errors.Trace(err)
					}

					entities := make([]*entity, 0, len(res.Entities))
					for _, e := range res.Entities {
						if e.ID == org.ID {
							for _, em := range e.Members {
								if em.Type == directory.EntityType_INTERNAL {
									ent, err := transformEntityToResponse(em)
									if err != nil {
										return nil, internalError(err)
									}
									entities = append(entities, ent)
								}
							}
						}
					}
					return entities, nil
				},
			},
			"savedThreadQueries": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(savedThreadQueryType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*organization)
					if org.Entity == nil || org.Entity.ID == "" {
						return nil, errors.New("no entity for organization")
					}
					svc := serviceFromParams(p)
					ctx := p.Context
					res, err := svc.threading.SavedQueries(ctx, &threading.SavedQueriesRequest{
						EntityID: org.Entity.ID,
					})
					if err != nil {
						return nil, internalError(err)
					}
					var qs []*savedThreadQuery
					for _, q := range res.SavedQueries {
						qs = append(qs, &savedThreadQuery{
							ID:             q.ID,
							OrganizationID: org.ID,
							// TODO: query
						})
					}
					return qs, nil
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*organization)
					svc := serviceFromParams(p)
					return deeplink.OrgURL(svc.webDomain, org.ID), nil
				},
			},
		},
	},
)
