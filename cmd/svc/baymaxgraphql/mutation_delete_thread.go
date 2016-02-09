package main

import (
	"github.com/sprucehealth/backend/libs/errors"
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
			return nil, errNotAuthenticated
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
				return nil, errors.New("thread not found")
			}
			return nil, internalError(err)
		}

		ent, err := svc.entityForAccountID(ctx, tres.Thread.OrganizationID, acc.ID)
		if err != nil {
			return nil, internalError(err)
		}
		if ent == nil {
			return nil, errors.New("not a member of the organization")
		}

		if _, err := svc.threading.DeleteThread(ctx, &threading.DeleteThreadRequest{
			ThreadID:      threadID,
			ActorEntityID: ent.ID,
		}); err != nil {
			return nil, internalError(err)
		}

		return &deleteThreadOutput{
			ClientMutationID: mutationID,
		}, nil
	},
}
