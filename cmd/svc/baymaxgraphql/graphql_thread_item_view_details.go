package main

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/graphql"
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
					ctx := p.Context
					tivd := p.Source.(*models.ThreadItemViewDetails)
					if tivd == nil {
						return nil, errors.InternalError(ctx, errors.New("thread item view details is nil"))
					}
					if selectingOnlyID(p) {
						return &models.Entity{ID: tivd.ActorEntityID}, nil
					}

					ram := raccess.ResourceAccess(p)
					e, err := ram.Entity(ctx, tivd.ActorEntityID, nil, 0)
					if err != nil {
						return nil, err
					}
					ent, err := transformEntityToResponse(e)
					if err != nil {
						return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %s", err))
					}
					return ent, nil
				},
			},
		},
	},
)

func lookupThreadItemViewDetails(ctx context.Context, ram raccess.ResourceAccessor, threadItemID string) ([]interface{}, error) {
	tivd, err := ram.ThreadItemViewDetails(ctx, threadItemID)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	resps, err := transformThreadItemViewDetailsToResponse(tivd)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	iResps := make([]interface{}, len(resps))
	for i, v := range resps {
		iResps[i] = v
	}
	return iResps, err
}
