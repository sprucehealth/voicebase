package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type deleteThreadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var deleteThreadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeleteThreadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var deleteThreadErrorCodeEnum = graphql.String

var deleteThreadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeleteThreadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: deleteThreadErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deleteThreadOutput)
			return ok
		},
	},
)

var deleteThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteThreadInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID := input["threadID"].(string)

		// Make sure thread exists (wasn't deleted) and get organization ID to be able to fetch entity for the account
		thread, err := ram.Thread(ctx, threadID, "")
		if err != nil {
			switch grpc.Code(err) {
			case codes.NotFound:
				return nil, errors.UserError(ctx, errors.ErrTypeNotFound, "Thread does not exist.")
			}
			return nil, errors.InternalError(ctx, err)
		}

		ent, err := ram.EntityForAccountID(ctx, thread.OrganizationID, acc.ID)
		if err != nil {
			return nil, err
		}
		if ent == nil || ent.Type != directory.EntityType_INTERNAL {
			return nil, errors.UserError(ctx, errors.ErrTypeNotAuthorized, "Permission denied.")
		}

		if err := ram.DeleteThread(ctx, threadID, ent.ID); err != nil {
			return nil, err
		}

		return &deleteThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
