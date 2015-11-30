package main

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

var errNotAuthenticated = errors.New("not authenticated")

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type: accountType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					acc := accountFromContext(contextFromParams(p))
					if acc == nil {
						return nil, errNotAuthenticated
					}
					return acc, nil
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
				Type: organizationType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := contextFromParams(p)
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated
					}

					orgID := p.Args["id"].(string)

					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: orgID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 0,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if err != nil {
						return nil, internalError(err)
					}
					if !res.Success {
						switch res.Failure.Reason {
						case directory.LookupEntitiesResponse_Failure_NOT_FOUND:
							return nil, errors.New("organization not found")
						}
						return nil, internalError(fmt.Errorf("Failed to get organization: %s %s", res.Failure.Reason, res.Failure.Message))
					}
					for _, em := range res.Entities {
						oc, err := transformContactsToResponse(em.Contacts)
						if err != nil {
							return nil, internalError(fmt.Errorf("failed to transform org contacts: %+v", err))
						}
						return &organization{
							ID:       em.ID,
							Name:     em.Name,
							Contacts: oc,
						}, nil
					}
					return nil, errors.New("organization not found")
				},
			},
			"savedThreadQuery": &graphql.Field{
				Type: savedThreadQueryType,
				Args: graphql.FieldConfigArgument{
					// "orgID":        &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := contextFromParams(p)
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated
					}

					// idArg := p.Args["id"].(string)
					// id := FromGlobalID(idArg)
					// if id == nil || id.Type != savedThreadQueryIDType {
					// 	return nil, errors.New("invalid saved thread query ID " + idArg)
					// }
					id := p.Args["id"].(string)

					tres, err := svc.threading.SavedQuery(ctx, &threading.SavedQueryRequest{
						SavedQueryID: id,
					})
					if err != nil {
						switch grpc.Code(err) {
						case codes.NotFound:
							return nil, err
						}
						return nil, internalError(err)
					}

					sq, err := transformSavedQueryToResponse(tres.SavedQuery)
					if err != nil {
						return nil, internalError(err)
					}
					return sq, nil
				},
			},
			"thread": &graphql.Field{
				Type: threadType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ctx := contextFromParams(p)
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated
					}

					// idArg := p.Args["id"].(string)
					// threadID := FromGlobalID(idArg)
					// if threadID == nil || threadID.Type != threadIDType {
					// 	return nil, errors.New("invalid thread ID " + idArg)
					// }
					id := p.Args["id"].(string)

					tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
						ThreadID: id,
					})
					if err != nil {
						switch grpc.Code(err) {
						case codes.NotFound:
							return nil, err
						}
						return nil, internalError(err)
					}

					thread, err := transformThreadToResponse(tres.Thread)
					if err != nil {
						return nil, internalError(err)
					}
					return thread, nil
				},
			},
		},
	},
)
