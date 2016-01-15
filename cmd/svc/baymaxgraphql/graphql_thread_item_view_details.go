package main

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

var threadItemViewDetailsType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ThreadItemViewDetails",
		Fields: graphql.Fields{
			"threadItemID":  &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"actorEntityID": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"viewTime":      &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"actor": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					tivd := p.Source.(*threadItemViewDetails)
					if tivd == nil {
						return nil, internalError(errors.New("thread item view details is nil"))
					}
					if selectingOnlyID(p) {
						return &entity{ID: tivd.ActorEntityID}, nil
					}

					svc := serviceFromParams(p)
					ctx := p.Context
					res, err := svc.directory.LookupEntities(ctx,
						&directory.LookupEntitiesRequest{
							LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
							LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
								EntityID: tivd.ActorEntityID,
							},
						})
					if err != nil {
						return nil, internalError(err)
					}
					for _, e := range res.Entities {
						return &entity{
							ID:   e.ID,
							Name: e.Name,
						}, nil
					}
					return nil, errors.New("actor not found")
				},
			},
		},
	},
)

func lookupThreadItemViewDetails(ctx context.Context, svc *service, threadItemID string) ([]interface{}, error) {
	res, err := svc.threading.ThreadItemViewDetails(ctx, &threading.ThreadItemViewDetailsRequest{
		ItemID: threadItemID,
	})
	if err != nil {
		return nil, internalError(err)
	}
	resps, err := transformThreadItemViewDetailsToResponse(res.ItemViewDetails)
	if err != nil {
		return nil, internalError(err)
	}
	iResps := make([]interface{}, len(resps))
	for i, v := range resps {
		iResps[i] = v
	}
	return iResps, err
}
