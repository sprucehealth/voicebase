package main

import (
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var leaveThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "LeaveThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"memberEntityIDs":  &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"title":            &graphql.InputObjectFieldConfig{Type: graphql.String},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var leaveThreadErrorCodeEnum = graphql.String

var leaveThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "LeaveThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: leaveThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*leaveThreadOutput)
		return ok
	},
})

type leaveThreadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var leaveThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(leaveThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(leaveThreadInputType)},
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

		thread, err := ram.Thread(ctx, threadID, "")
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if thread == nil {
			return nil, errors.ErrNotFound(ctx, threadID)
		}
		if thread.Type != threading.ThreadType_TEAM {
			return nil, errors.New("Cannot leave non-team threads")
		}

		ent, err := ram.EntityForAccountID(ctx, thread.OrganizationID, acc.ID)
		if err != nil {
			return nil, err
		}

		members, err := ram.ThreadMembers(ctx, thread.OrganizationID, &threading.ThreadMembersRequest{ThreadID: thread.ID})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		names := make([]string, 0, len(members))
		for _, m := range members {
			if m.ID != ent.ID {
				names = append(names, m.Info.DisplayName)
			}
		}

		_, err = ram.UpdateThread(ctx, &threading.UpdateThreadRequest{
			SystemTitle:           strings.Join(names, ", "),
			RemoveMemberEntityIDs: []string{ent.ID},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &leaveThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
