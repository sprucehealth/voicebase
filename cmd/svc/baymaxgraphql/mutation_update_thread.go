package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var updateThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"memberEntityIDs":  &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"title":            &graphql.InputObjectFieldConfig{Type: graphql.String},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var updateThreadErrorCodeEnum = graphql.String

var updateThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*updateThreadOutput)
		return ok
	},
})

type updateThreadOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Thread           *models.Thread `json:"thread"`
}

var updateThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateThreadInputType)},
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
			return nil, errors.New("Cannot modify non-team threads")
		}

		updateReq := &threading.UpdateThreadRequest{
			ThreadID: threadID,
		}
		if t, ok := input["title"].(string); ok {
			updateReq.UserTitle = t
		}
		if ms, ok := input["memberEntityIDs"].([]interface{}); ok && len(ms) != 0 {
			members := make([]string, len(ms))
			for i, m := range ms {
				members[i] = m.(string)
			}
			members, systemTitle, err := teamThreadMembersAndTitle(ctx, ram, thread.OrganizationID, members)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if len(members) != 0 {
				updateReq.SystemTitle = systemTitle
				updateReq.SetMemberEntityIDs = members
			}
		}

		if updateReq.UserTitle != "" || updateReq.SystemTitle != "" || len(updateReq.SetMemberEntityIDs) != 0 {
			res, err := ram.UpdateThread(ctx, updateReq)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			thread = res.Thread
		}

		th, err := transformThreadToResponse(thread)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &updateThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Thread:           th,
		}, nil
	},
}
