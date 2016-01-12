package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
					ctx := p.Context
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

func lookupAccount(ctx context.Context, svc *service, id string) (interface{}, error) {
	res, err := svc.auth.GetAccount(ctx, &auth.GetAccountRequest{
		AccountID: id,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, errors.New("account not found")
		}
		return nil, internalError(err)
	}
	// Since we only use the ID we don't really need to do the lookup, but
	// it allows us to check if the account exists.
	return &account{
		ID: res.Account.ID,
	}, nil
}
