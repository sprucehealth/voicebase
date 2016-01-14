package main

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
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
			"id":                   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":                &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subtitle":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"lastMessageTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"primaryEntity": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					th := p.Source.(*thread)
					if th == nil {
						return nil, internalError(errors.New("thread is nil"))
					}
					if selectingOnlyID(p) {
						return &entity{ID: th.PrimaryEntityID}, nil
					}

					svc := serviceFromParams(p)
					ctx := p.Context
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
						return nil, internalError(err)
					}
					for _, e := range res.Entities {
						cs, err := transformContactsToResponse(e.Contacts)
						if err != nil {
							return nil, internalError(fmt.Errorf("failed to transform contacts: %s", err))
						}
						return &entity{
							ID:       e.ID,
							Name:     e.Name,
							Contacts: cs,
						}, nil
					}
					return nil, errors.New("primary entity not found")
				},
			},
			"items": &graphql.Field{
				Type: graphql.NewNonNull(threadItemConnectionType.ConnectionType),
				Args: NewConnectionArguments(nil),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					t := p.Source.(*thread)
					if t == nil {
						return nil, internalError(errors.New("thread is nil"))
					}
					svc := serviceFromParams(p)
					ctx := p.Context
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
					for i, e := range res.Edges {
						it, err := transformThreadItemToResponse(e.Item, "")
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
			// "members": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(memberType))},
		},
	},
)

func lookupThread(ctx context.Context, svc *service, id string) (interface{}, error) {
	tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
		ThreadID: id,
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, errors.New("thread not found")
		}
		return nil, internalError(err)
	}

	th, err := transformThreadToResponse(tres.Thread)
	if err != nil {
		return nil, internalError(err)
	}
	if err := svc.hydrateThreadTitles(ctx, []*thread{th}); err != nil {
		return nil, internalError(err)
	}
	return th, nil
}
