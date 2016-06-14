package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/graphql"
)

var pendingCallsQuery = &graphql.Field{
	Type: graphql.NewNonNull(graphql.NewList(callType)),
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		ram := raccess.ResourceAccess(p)
		res, err := ram.PendingIPCalls(ctx)
		if err != nil {
			return nil, err
		}
		calls := make([]*models.Call, len(res.Calls))
		for i, c := range res.Calls {
			call, err := transformCallToResponse(c, acc.ID)
			if err != nil {
				return nil, err
			}
			calls[i] = call
		}
		return calls, nil
	}),
}
