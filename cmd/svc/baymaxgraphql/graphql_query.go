package main

import (
	"errors"
	"strings"

	"github.com/graphql-go/graphql"
)

var errNotAuthenticated = errors.New("not authenticated")

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type: graphql.NewNonNull(accountType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					acc := accountFromContext(p.Context)
					if acc == nil {
						return nil, errNotAuthenticated
					}
					return acc, nil
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
						return nil, errNotAuthenticated
					}
					id := p.Args["id"].(string)
					if strings.HasPrefix(id, "entity:") {
						return lookupEntity(ctx, svc, id)
					} else if strings.HasPrefix(id, "account:") {
						if id == acc.ID {
							return acc, nil
						}
						return lookupAccount(ctx, svc, id)
					} else {
						i := strings.IndexByte(id, '_')
						prefix := id[:i]
						switch prefix {
						case "sq":
							return lookupSavedQuery(ctx, svc, id)
						case "t":
							return lookupThread(ctx, svc, id)
						case "ti":
							return lookupThreadItem(ctx, svc, id)
						}
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
						return nil, errNotAuthenticated
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
						return nil, errNotAuthenticated
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
						return nil, errNotAuthenticated
					}
					id := p.Args["id"].(string)
					return lookupThread(ctx, svc, id)
				},
			},
		},
	},
)
