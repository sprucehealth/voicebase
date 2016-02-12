package main

import (
	"errors"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type: graphql.NewNonNull(meType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					acc := accountFromContext(p.Context)
					if acc == nil {
						return nil, errNotAuthenticated(p.Context)
					}
					cek := clientEncryptionKeyFromContext(p.Context)
					return &me{Account: acc, ClientEncryptionKey: cek}, nil
				},
			},
			"node": &graphql.Field{
				Type: graphql.NewNonNull(nodeInterfaceType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}
					id := p.Args["id"].(string)
					prefix := nodePrefix(id)
					switch prefix {
					case "entity":
						return lookupEntity(ctx, svc, id)
					case "account":
						if id == acc.ID {
							return acc, nil
						}
						return lookupAccount(ctx, svc, id)
					case "sq":
						return lookupSavedQuery(ctx, svc, id)
					case "t":
						return lookupThreadWithReadStatus(ctx, svc, acc, id)
					case "ti":
						return lookupThreadItem(ctx, svc, id)
					}
					return nil, errors.New("unknown node type")
				},
			},
			// "listSavedThreadQueries": &graphql.Field{
			// 	Type: graphql.NewList(graphql.NewNonNull(savedThreadQueryType)),
			// 	Args: graphql.FieldConfigArgument{
			// 		"orgID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			// 	},
			// 	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			// 		return nil, nil
			// 	},
			// },
			"organization": &graphql.Field{
				Type: graphql.NewNonNull(organizationType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}
					id := p.Args["id"].(string)
					return lookupEntity(ctx, svc, id)
				},
			},
			"savedThreadQuery": &graphql.Field{
				Type: graphql.NewNonNull(savedThreadQueryType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}
					id := p.Args["id"].(string)
					return lookupSavedQuery(ctx, svc, id)
				},
			},
			"thread": &graphql.Field{
				Type: graphql.NewNonNull(threadType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}
					it, err := lookupThreadWithReadStatus(ctx, svc, acc, p.Args["id"].(string))
					return it, err
				},
			},
			"subdomain": &graphql.Field{
				Type: graphql.NewNonNull(subdomainType),
				Args: graphql.FieldConfigArgument{
					"value": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					domain := p.Args["value"].(string)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}

					queriedEntityID, queriedDomain, err := svc.entityDomain(ctx, "", domain)
					if err != nil {
						return nil, err
					}

					return &subdomain{
						Available: queriedEntityID == "" && queriedDomain == "",
					}, nil
				},
			},
			"setting":            settingsQuery,
			"forceUpgradeStatus": forceUpgradeQuery,
		},
	},
)

// TODO: This double read is inefficent/incorrect in the sense that we need the org ID to get the correct entity. We will use this for now until we can encode the organization ID into the thread ID
func lookupThreadWithReadStatus(ctx context.Context, svc *service, acc *account, id string) (interface{}, error) {
	th, err := lookupThread(ctx, svc, id, "")
	if err != nil {
		return nil, internalError(ctx, err)
	}
	ent, err := svc.entityForAccountID(ctx, th.OrganizationID, acc.ID)
	if err != nil {
		return nil, internalError(ctx, err)
	}
	if ent == nil {
		golog.Debugf("Account %s not a member of organization %d", acc.ID, th.OrganizationID)
		return nil, userError(ctx, errTypeNotAuthorized, "You are not a member of the organzation")
	}
	return lookupThread(ctx, svc, id, ent.ID)
}
