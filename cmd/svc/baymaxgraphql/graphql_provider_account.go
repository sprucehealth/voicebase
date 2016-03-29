package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

var providerAccountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProviderAccount",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
			accountInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := p.Source.(*models.ProviderAccount)
					if acc == nil {
						// Shouldn't be possible I don't think
						return nil, errors.InternalError(ctx, errors.New("nil account"))
					}
					entities, err := ram.EntitiesForExternalID(ctx, acc.ID, []directory.EntityInformation{
						directory.EntityInformation_MEMBERSHIPS,
						directory.EntityInformation_CONTACTS,
					}, 1, nil)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					sh := gqlctx.SpruceHeaders(p.Context)

					var orgs []*models.Organization
					for _, e := range entities {
						for _, em := range e.Memberships {
							if em.Type == directory.EntityType_ORGANIZATION {
								oc, err := transformContactsToResponse(em.Contacts)
								if err != nil {
									return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform org contacts: %+v", err))
								}
								entity, err := transformEntityToResponse(svc.staticURLPrefix, e, sh)
								if err != nil {
									return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %+v", err))
								}
								orgs = append(orgs, &models.Organization{
									ID:       em.ID,
									Name:     em.Info.DisplayName,
									Contacts: oc,
									Entity:   entity,
								})
							}
						}
					}
					return orgs, nil
				},
			},
		},
	},
)
