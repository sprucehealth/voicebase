package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/graphql"
)

var markThreadAsReadMutation = &graphql.Field{
	Type: graphql.NewNonNull(markThreadAsReadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(markThreadAsReadInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID, _ := input["threadID"].(string)
		orgID, _ := input["organizationID"].(string)
		ent, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if err = ram.MarkThreadAsRead(ctx, threadID, ent.ID); err != nil {
			return nil, err
		}

		return &markThreadAsReadOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	}),
}
