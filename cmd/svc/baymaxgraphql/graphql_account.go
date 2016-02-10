package main

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var meType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Me",
		Fields: graphql.Fields{
			"account":             &graphql.Field{Type: graphql.NewNonNull(accountType)},
			"clientEncryptionKey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var accountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Account",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := p.Source.(*account)
					if acc == nil {
						// Shouldn't be possible I don't think
						return nil, internalError(ctx, errors.New("nil account"))
					}
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
								ExternalID: acc.ID,
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
						return nil, internalError(ctx, err)
					}
					var orgs []*organization
					for _, e := range res.Entities {
						for _, em := range e.Memberships {
							oc, err := transformContactsToResponse(em.Contacts)
							if err != nil {
								return nil, internalError(ctx, fmt.Errorf("failed to transform org contacts: %+v", err))
							}
							entity, err := transformEntityToResponse(e)
							if err != nil {
								return nil, internalError(ctx, fmt.Errorf("failed to transform entity: %+v", err))
							}
							orgs = append(orgs, &organization{
								ID:       em.ID,
								Name:     em.Info.DisplayName,
								Contacts: oc,
								Entity:   entity,
							})
						}
					}
					return orgs, nil
				},
			},
		},
	},
)

func lookupAccount(ctx context.Context, svc *service, id string) (interface{}, error) {
	res, err := svc.auth.GetAccount(ctx, &auth.GetAccountRequest{
		AccountID: id,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, userError(ctx, errTypeNotFound, "Account not found.")
		}
		return nil, internalError(ctx, err)
	}
	// Since we only use the ID we don't really need to do the lookup, but
	// it allows us to check if the account exists.
	return &account{
		ID: res.Account.ID,
	}, nil
}
