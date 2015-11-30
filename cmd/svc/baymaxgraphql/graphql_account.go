package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
)

var accountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Account",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)}, //GlobalIDField(accountIDType),
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := contextFromParams(p)
					acc := p.Source.(*account)
					if acc == nil {
						// Shouldn't be possible I don't think
						return nil, internalError(errors.New("nil account"))
					}
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
								ExternalID: accountIDType + ":" + acc.ID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 1,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_MEMBERSHIPS,
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if err != nil {
						return nil, internalError(err)
					}
					if !res.Success {
						// Should never fail
						return nil, internalError(fmt.Errorf("Failed to get account memberships: %s %s", res.Failure.Reason, res.Failure.Message))
					}
					var orgs []*organization
					for _, e := range res.Entities {
						for _, em := range e.Memberships {
							oc, err := transformContactsToResponse(em.Contacts)
							if err != nil {
								return nil, internalError(fmt.Errorf("failed to transform org contacts: %+v", err))
							}
							ec, err := transformContactsToResponse(e.Contacts)
							if err != nil {
								return nil, internalError(fmt.Errorf("failed to transform entity contacts: %+v", err))
							}
							orgs = append(orgs, &organization{
								ID:       em.ID,
								Name:     em.Name,
								Contacts: oc,
								Entity: &entity{
									ID:       e.ID,
									Name:     e.Name,
									Contacts: ec,
								},
							})
						}
					}
					return orgs, nil
				},
			},
		},
	},
)
