package main

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var threadConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "Thread",
	NodeType: threadType,
})

var threadType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Thread",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":                    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":                 &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subtitle":              &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"lastMessageTimestamp":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"unread":                &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowInternalMessages": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			// TODO: We currently just assume all contacts for an entity are available endpoints
			"availableEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*thread)
					if th == nil {
						return nil, internalError(ctx, errors.New("thread is nil"))
					}

					svc := serviceFromParams(p)
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: th.PrimaryEntityID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 0,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if err != nil {
						return nil, internalError(ctx, err)
					}
					for _, e := range res.Entities {
						endpoints := make([]*endpoint, len(e.Contacts))
						for i, c := range e.Contacts {
							endpoint, err := transformEntityContactToEndpoint(c)
							if err != nil {
								return nil, internalError(ctx, err)
							}
							endpoints[i] = endpoint
						}
						return endpoints, nil
					}
					return nil, errors.New("primary entity not found")
				},
			},
			// Default endpoints are build from the last primary entity endpoints filtering out anything contacts that no longer exist for the entity
			"defaultEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*thread)
					if th == nil {
						return nil, internalError(ctx, errors.New("thread is nil"))
					}

					svc := serviceFromParams(p)
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: th.PrimaryEntityID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 0,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if err != nil {
						return nil, internalError(ctx, err)
					}

					for _, e := range res.Entities {
						var filteredEndpoints []*endpoint
						// Assert that our endpoints stil exist as a contact
						for _, ep := range th.LastPrimaryEntityEndpoints {
							for _, c := range e.Contacts {
								endpoint, err := transformEntityContactToEndpoint(c)
								if err != nil {
									return nil, internalError(ctx, err)
								}
								if endpoint.Channel == ep.Channel && endpoint.ID == ep.ID {
									filteredEndpoints = append(filteredEndpoints, endpoint)
									continue
								}
							}
						}
						// If we didn't find any matching endpoints or the source list is empty, pick the first contact attached to the entity
						if len(filteredEndpoints) == 0 {
							for _, c := range e.Contacts {
								endpoint, err := transformEntityContactToEndpoint(c)
								if err != nil {
									return nil, internalError(ctx, err)
								}
								filteredEndpoints = append(filteredEndpoints, endpoint)
								continue
							}
						}
						return filteredEndpoints, nil
					}
					return nil, errors.New("primary entity not found")
				},
			},
			"primaryEntity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*thread)
					if th == nil {
						return nil, internalError(ctx, errors.New("thread is nil"))
					}
					// Internal threads don't have a primary entity
					if th.PrimaryEntityID == "" {
						return nil, nil
					}
					if selectingOnlyID(p) {
						return &entity{ID: th.PrimaryEntityID}, nil
					}

					svc := serviceFromParams(p)
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: th.PrimaryEntityID,
							},
							RequestedInformation: &directory.RequestedInformation{
								Depth: 0,
								EntityInformation: []directory.EntityInformation{
									directory.EntityInformation_CONTACTS,
								},
							},
						})
					if err != nil {
						return nil, internalError(ctx, err)
					}
					for _, e := range res.Entities {
						ent, err := transformEntityToResponse(e)
						if err != nil {
							return nil, internalError(ctx, fmt.Errorf("failed to transform entity: %s", err))
						}
						return ent, nil
					}
					return nil, internalError(ctx, errors.New("Primary entity not found"))
				},
			},
			"items": &graphql.Field{
				Type: graphql.NewNonNull(threadItemConnectionType.ConnectionType),
				Args: NewConnectionArguments(nil),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					t := p.Source.(*thread)
					if t == nil {
						return nil, internalError(ctx, errors.New("thread is nil"))
					}
					svc := serviceFromParams(p)
					acc := accountFromContext(p.Context)
					if acc == nil {
						return nil, errNotAuthenticated(ctx)
					}

					req := &threading.ThreadItemsRequest{
						ThreadID: t.ID,
						// TODO: ViewerEntityID
						Iterator: &threading.Iterator{},
					}
					if s, ok := p.Args["after"].(string); ok {
						req.Iterator.StartCursor = s
					}
					if s, ok := p.Args["before"].(string); ok {
						req.Iterator.EndCursor = s
					}
					if i, ok := p.Args["last"].(int); ok {
						req.Iterator.Count = uint32(i)
						req.Iterator.Direction = threading.Iterator_FROM_END
					} else if i, ok := p.Args["first"].(int); ok {
						req.Iterator.Count = uint32(i)
						req.Iterator.Direction = threading.Iterator_FROM_START
					} else {
						req.Iterator.Count = 20 // default
						req.Iterator.Direction = threading.Iterator_FROM_START
					}
					res, err := svc.threading.ThreadItems(ctx, req)
					if err != nil {
						switch grpc.Code(err) {
						case codes.NotFound:
							return nil, err
						case codes.InvalidArgument:
							return nil, err
						}
						return nil, internalError(ctx, err)
					}

					cn := &Connection{
						Edges: make([]*Edge, len(res.Edges)),
					}
					if req.Iterator.Direction == threading.Iterator_FROM_START {
						cn.PageInfo.HasNextPage = res.HasMore
					} else {
						cn.PageInfo.HasPreviousPage = res.HasMore
					}

					for i, e := range res.Edges {
						it, err := transformThreadItemToResponse(e.Item, "", acc.ID, svc.mediaSigner)
						if err != nil {
							golog.Errorf("Unknown thread item type %s", e.Item.Type.String())
							continue
						}
						cn.Edges[i] = &Edge{
							Node:   it,
							Cursor: ConnectionCursor(e.Cursor),
						}
					}

					return cn, nil
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Args: graphql.FieldConfigArgument{
					"savedQueryID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*thread)
					svc := serviceFromParams(p)
					savedQueryID, _ := p.Args["savedQueryID"].(string)
					return deeplink.ThreadURL(svc.webDomain, th.OrganizationID, savedQueryID, th.ID), nil
				},
			},
			"shareableDeeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*thread)
					svc := serviceFromParams(p)
					return deeplink.ThreadURLShareable(svc.webDomain, th.OrganizationID, th.ID), nil
				},
			},
			// "members": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(memberType))},
		},
	},
)

func lookupThread(ctx context.Context, svc *service, id, viewerEntityID string) (interface{}, error) {
	tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
		ThreadID:       id,
		ViewerEntityID: viewerEntityID,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, userError(ctx, errTypeNotFound, "Thread not found.")
		}
		return nil, internalError(ctx, err)
	}

	th, err := transformThreadToResponse(tres.Thread)
	if err != nil {
		return nil, internalError(ctx, err)
	}
	if err := svc.hydrateThreadTitles(ctx, []*thread{th}); err != nil {
		return nil, internalError(ctx, err)
	}
	return th, nil
}
