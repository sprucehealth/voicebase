package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var savedThreadQueryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SavedThreadQuery",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			// TODO: query
			"threads": &graphql.Field{
				Type: threadConnectionType.ConnectionType,
				Args: NewConnectionArguments(nil),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					stq := p.Source.(*models.SavedThreadQuery)
					if stq == nil {
						// Shouldn't be possible I don't think
						return nil, errors.InternalError(ctx, errors.New("savedThreadQuery is nil"))
					}

					ram := raccess.ResourceAccess(p)
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}

					ent, err := entityInOrgForAccountID(ctx, ram, stq.OrganizationID, acc)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}
					if ent == nil || ent.Type != directory.EntityType_INTERNAL {
						return nil, errors.UserError(ctx, errors.ErrTypeNotAuthorized, "Not a member of the organization")
					}
					req := &threading.QueryThreadsRequest{
						OrganizationID: stq.OrganizationID,
						Type:           threading.QueryThreadsRequest_SAVED,
						QueryType: &threading.QueryThreadsRequest_SavedQueryID{
							SavedQueryID: stq.ID,
						},
						Iterator:       &threading.Iterator{},
						ViewerEntityID: ent.ID,
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
					}
					res, err := ram.QueryThreads(ctx, req)
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
					threads := make([]*models.Thread, len(res.Edges))
					for i, e := range res.Edges {
						t, err := transformThreadToResponse(ctx, ram, e.Thread, acc)
						if err != nil {
							return nil, errors.InternalError(ctx, fmt.Errorf("Failed to transform thread: %s", err))
						}
						threads[i] = t
						cn.Edges[i] = &Edge{
							Node:   t,
							Cursor: ConnectionCursor(e.Cursor),
						}
					}

					if err := hydrateThreads(ctx, ram, threads); err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					return cn, nil
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					sq := p.Source.(*models.SavedThreadQuery)
					svc := serviceFromParams(p)
					return deeplink.SavedQueryURL(svc.webDomain, sq.OrganizationID, sq.ID), nil
				},
			},
		},
	},
)

func lookupSavedQuery(ctx context.Context, ram raccess.ResourceAccessor, savedQueryID string) (interface{}, error) {
	sq, err := ram.SavedQuery(ctx, savedQueryID)
	if err != nil {
		return nil, err
	}

	rsq, err := transformSavedQueryToResponse(sq)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return rsq, nil
}
