package main

import (
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var deleteThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteThreadInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID := input["threadID"].(string)

		// Make sure thread exists (wasn't deleted) and get organization ID to be able to fetch entity for the account
		tres, err := svc.threading.Thread(ctx, &threading.ThreadRequest{
			ThreadID: threadID,
		})
		if err != nil {
			switch grpc.Code(err) {
			case codes.NotFound:
				return nil, userError(ctx, errTypeNotFound, "Thread does not exist.")
			}
			return nil, internalError(ctx, err)
		}

		ent, err := svc.entityForAccountID(ctx, tres.Thread.OrganizationID, acc.ID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if ent == nil || ent.Type != directory.EntityType_INTERNAL {
			return nil, userError(ctx, errTypeNotAuthorized, "Permission denied.")
		}

		if _, err := svc.threading.DeleteThread(ctx, &threading.DeleteThreadRequest{
			ThreadID:      threadID,
			ActorEntityID: ent.ID,
		}); err != nil {
			return nil, internalError(ctx, err)
		}

		return &deleteThreadOutput{
			ClientMutationID: mutationID,
		}, nil
	},
}
