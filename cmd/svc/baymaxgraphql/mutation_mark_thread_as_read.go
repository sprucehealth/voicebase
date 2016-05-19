package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
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

		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
			},
			Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		}, orgID)
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
