package main

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/svc/notification/deeplink"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
					stq := p.Source.(*savedThreadQuery)
					if stq == nil {
						// Shouldn't be possible I don't think
						return nil, internalError(errors.New("savedThreadQuery is nil"))
					}

					svc := serviceFromParams(p)
					ctx := p.Context
					acc := accountFromContext(ctx)
					if acc == nil {
						return nil, errNotAuthenticated
					}
					ent, err := svc.entityForAccountID(ctx, stq.OrganizationID, acc.ID)
					if err != nil || ent == nil {
						return nil, internalError(errors.New("no entity id found"))
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
					res, err := svc.threading.QueryThreads(ctx, req)
					if err != nil {
						switch grpc.Code(err) {
						case codes.InvalidArgument:
							return nil, err
						}
						return nil, internalError(err)
					}

					cn := &Connection{
						Edges: make([]*Edge, len(res.Edges)),
					}
					if req.Iterator.Direction == threading.Iterator_FROM_START {
						cn.PageInfo.HasNextPage = res.HasMore
					} else {
						cn.PageInfo.HasPreviousPage = res.HasMore
					}
					threads := make([]*thread, len(res.Edges))
					for i, e := range res.Edges {
						t, err := transformThreadToResponse(e.Thread)
						if err != nil {
							return nil, internalError(fmt.Errorf("Failed to transform thread: %s", err))
						}
						threads[i] = t
						cn.Edges[i] = &Edge{
							Node:   t,
							Cursor: ConnectionCursor(e.Cursor),
						}
					}
					if err := svc.hydrateThreadTitles(ctx, threads); err != nil {
						return nil, internalError(err)
					}

					return cn, nil
				},
			},
			"deeplink": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					sq := p.Source.(*savedThreadQuery)
					svc := serviceFromParams(p)
					return deeplink.SavedQueryURL(svc.webDomain, sq.OrganizationID, sq.ID), nil
				},
			},
		},
	},
)

func lookupSavedQuery(ctx context.Context, svc *service, id string) (interface{}, error) {
	tres, err := svc.threading.SavedQuery(ctx, &threading.SavedQueryRequest{
		SavedQueryID: id,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, errors.New("saved query not found")
		}
		return nil, internalError(err)
	}

	sq, err := transformSavedQueryToResponse(tres.SavedQuery)
	if err != nil {
		return nil, internalError(err)
	}
	return sq, nil
}
