package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"

	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
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
			"allowAddMembers":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowDelete":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowEmailAttachments": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowInternalMessages": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowLeave":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowRemoveMembers":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowSMSAttachments":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowUpdateTitle":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"allowExternalDelivery": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"emptyStateTextMarkup":  &graphql.Field{Type: graphql.String},
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"lastMessageTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"subtitle":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"title":                &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"unread":               &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"unreadReference":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"isDeletable": &graphql.Field{
				Type:              graphql.NewNonNull(graphql.Boolean),
				DeprecationReason: "Replaced with allowDelete",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*models.Thread).AllowDelete, nil
				},
			},
			"members": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// Only team threads have members
					if th.Type != models.ThreadTypeTeam {
						return nil, nil
					}

					svc := serviceFromParams(p)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					ram := raccess.ResourceAccess(p)
					members, err := ram.ThreadMembers(ctx, th.OrganizationID, &threading.ThreadMembersRequest{
						ThreadID: th.ID,
					})
					if err != nil {
						return nil, err
					}
					sh := gqlctx.SpruceHeaders(ctx)
					ms := make([]*models.Entity, len(members))
					for i, em := range members {
						e, err := transformEntityToResponse(svc.staticURLPrefix, em, sh, acc)
						if err != nil {
							return nil, err
						}
						ms[i] = e
					}
					return ms, nil
				},
			},
			"addressableEntities": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(entityType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}

					svc := serviceFromParams(p)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					ram := raccess.ResourceAccess(p)

					switch th.Type {
					case models.ThreadTypeTeam:
						members, err := ram.ThreadMembers(ctx, th.OrganizationID, &threading.ThreadMembersRequest{
							ThreadID: th.ID,
						})
						if err != nil {
							return nil, err
						}
						ms := make([]*models.Entity, len(members))
						for i, em := range members {
							e, err := transformEntityToResponse(svc.staticURLPrefix, em, gqlctx.SpruceHeaders(ctx), acc)
							if err != nil {
								return nil, err
							}
							ms[i] = e
						}
						return ms, nil
					case models.ThreadTypeExternal:
						orgEntity, err := ram.Entity(ctx, th.OrganizationID, []directory.EntityInformation{
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
								ent, err := transformEntityToResponse(svc.staticURLPrefix, em, gqlctx.SpruceHeaders(ctx), acc)
								if err != nil {
									return nil, errors.InternalError(ctx, err)
								}
								entities = append(entities, ent)
							}
						}
						return entities, nil
					}
					return nil, nil
				},
			},
			// TODO: We currently just assume all contacts for an entity are available endpoints
			"availableEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// No endpoints for team threads
					if th.Type == models.ThreadTypeTeam || th.Type == models.ThreadTypeSecureExternal {
						return nil, nil
					}

					ram := raccess.ResourceAccess(p)
					ent, err := primaryEntityForThread(ctx, ram, th)
					if err != nil {
						return nil, err
					}
					if ent.Type != directory.EntityType_EXTERNAL {
						return []*models.Endpoint{}, nil
					}

					endpoints := make([]*models.Endpoint, len(ent.Contacts))
					for i, c := range ent.Contacts {
						endpoint, err := transformEntityContactToEndpoint(c)
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}
						endpoints[i] = endpoint
					}
					return endpoints, nil
				},
			},
			// Default endpoints are build from the last primary entity endpoints filtering out anything contacts that no longer exist for the entity
			"defaultEndpoints": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(endpointType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// No endpoints for team threads
					if th.Type == models.ThreadTypeTeam || th.Type == models.ThreadTypeSecureExternal {
						return nil, nil
					}

					ram := raccess.ResourceAccess(p)
					ent, err := primaryEntityForThread(ctx, ram, th)
					if err != nil {
						return nil, err
					}
					if ent.Type != directory.EntityType_EXTERNAL {
						return []*models.Endpoint{}, nil
					}

					var filteredEndpoints []*models.Endpoint
					// Assert that our endpoints still exist as a contact
					for _, ep := range th.LastPrimaryEntityEndpoints {
						for _, c := range ent.Contacts {
							endpoint, err := transformEntityContactToEndpoint(c)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							if endpoint.Channel == ep.Channel && endpoint.ID == ep.ID {
								filteredEndpoints = append(filteredEndpoints, endpoint)
								break
							}
						}
					}
					// If we didn't find any matching endpoints or the source list is empty, pick the first contact attached to the entity
					if len(filteredEndpoints) == 0 {
						for _, c := range ent.Contacts {
							endpoint, err := transformEntityContactToEndpoint(c)
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							filteredEndpoints = append(filteredEndpoints, endpoint)
							break
						}
					}
					return filteredEndpoints, nil
				},
			},
			"primaryEntity": &graphql.Field{
				Type: entityType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					svc := serviceFromParams(p)
					th := p.Source.(*models.Thread)
					if th == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					// Internal threads don't have a primary entity
					if th.PrimaryEntityID == "" {
						// TODO: for now returning a stub primary entity as apps are relying on it existing. remove at some point
						return stubEntity, nil
					}
					if selectingOnlyID(p) {
						return &models.Entity{ID: th.PrimaryEntityID}, nil
					}

					ram := raccess.ResourceAccess(p)
					pe, err := primaryEntityForThread(ctx, ram, th)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					sh := gqlctx.SpruceHeaders(ctx)
					ent, err := transformEntityToResponse(svc.staticURLPrefix, pe, sh, gqlctx.Account(ctx))
					if err != nil {
						return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %s", err))
					}
					return ent, nil
				},
			},
			"items": &graphql.Field{
				Type: graphql.NewNonNull(threadItemConnectionType.ConnectionType),
				Args: NewConnectionArguments(nil),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					t := p.Source.(*models.Thread)
					if t == nil {
						return nil, errors.InternalError(ctx, errors.New("thread is nil"))
					}
					svc := serviceFromParams(p)
					ram := raccess.ResourceAccess(p)
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
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
					res, err := ram.ThreadItems(ctx, req)
					if err != nil {
						return nil, err
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
							golog.Errorf("Failed to transform thread item %s: %s", e.Item.ID, err)
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
					th := p.Source.(*models.Thread)
					svc := serviceFromParams(p)
					savedQueryID, _ := p.Args["savedQueryID"].(string)
					return deeplink.ThreadURL(svc.webDomain, th.OrganizationID, savedQueryID, th.ID), nil
				},
			},
			"shareableDeeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*models.Thread)
					svc := serviceFromParams(p)
					return deeplink.ThreadURLShareable(svc.webDomain, th.OrganizationID, th.ID), nil
				},
			},
		},
	},
)

func lookupThread(ctx context.Context, ram raccess.ResourceAccessor, threadID, viewerEntityID string) (*models.Thread, error) {
	thread, err := ram.Thread(ctx, threadID, viewerEntityID)
	if err != nil {
		return nil, err
	}

	th, err := transformThreadToResponse(thread, gqlctx.Account(ctx))
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return th, nil
}

func primaryEntityForThread(ctx context.Context, ram raccess.ResourceAccessor, t *models.Thread) (*directory.Entity, error) {
	t.Mu.RLock()
	if t.PrimaryEntity != nil {
		t.Mu.RUnlock()
		return t.PrimaryEntity, nil
	}
	t.Mu.RUnlock()

	t.Mu.Lock()
	defer t.Mu.Unlock()

	// Could have been fetched by someone else at this point
	if t.PrimaryEntity != nil {
		return t.PrimaryEntity, nil
	}
	ent, err := ram.Entity(ctx, t.PrimaryEntityID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
	}, 0)
	t.PrimaryEntity = ent
	return ent, err
}
