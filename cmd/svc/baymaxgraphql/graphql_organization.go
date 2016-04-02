package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/settings"

	"github.com/sprucehealth/graphql"
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
			"allowTeamConversations": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Boolean),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					org := p.Source.(*models.Organization)
					if org == nil {
						return false, nil
					}

					booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
						NodeID: org.ID,
						Keys: []*settings.ConfigKey{
							{
								Key: baymaxgraphqlsettings.ConfigKeyTeamConversations,
							},
						},
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return booleanValue.Value, nil
				},
			},
			"entity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					if org.Entity != nil {
						return org.Entity, nil
					}

					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}

					e, err := ram.EntityForAccountID(ctx, org.ID, acc.ID)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					if e == nil {
						return nil, errors.New("entity not found for organization")
					}
					sh := gqlctx.SpruceHeaders(ctx)
					rE, err := transformEntityToResponse(svc.staticURLPrefix, e, sh)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					return rE, nil
				},
			},
			"contacts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(contactInfoType))},
			"entities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					if org.Entity == nil || org.Entity.ID == "" {
						return nil, errors.New("no entity for organization")
					}
					ram := raccess.ResourceAccess(p)
					svc := serviceFromParams(p)
					ctx := p.Context
					sh := gqlctx.SpruceHeaders(ctx)

					orgEntity, err := ram.Entity(ctx, org.ID, []directory.EntityInformation{
						directory.EntityInformation_MEMBERS,
						// TODO: don't always need contacts
						directory.EntityInformation_CONTACTS,
					}, 0)
					if err != nil {
						return nil, err
					}

					entities := make([]*models.Entity, 0, len(orgEntity.Members))
					for _, em := range orgEntity.Members {
						if em.Type == directory.EntityType_INTERNAL {
							ent, err := transformEntityToResponse(svc.staticURLPrefix, em, sh)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							entities = append(entities, ent)
						}
					}
					return entities, nil
				},
			},
			"savedThreadQueries": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(savedThreadQueryType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					org := p.Source.(*models.Organization)
					if org.Entity == nil || org.Entity.ID == "" {
						return nil, errors.New("no entity for organization")
					}
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					sqs, err := ram.SavedQueries(ctx, org.Entity.ID)
					if err != nil {
						return nil, err
					}
					var qs []*models.SavedThreadQuery
					for _, q := range sqs {
						qs = append(qs, &models.SavedThreadQuery{
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
					org := p.Source.(*models.Organization)
					svc := serviceFromParams(p)
					return deeplink.OrgURL(svc.webDomain, org.ID), nil
				},
			},
		},
	},
)
