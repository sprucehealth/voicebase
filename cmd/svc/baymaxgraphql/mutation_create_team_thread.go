package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var createTeamThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateTeamThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.String},
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"title":            &graphql.InputObjectFieldConfig{Type: graphql.String},
		"memberEntityIDs":  &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var createTeamThreadErrorCodeEnum = graphql.String

type createTeamThreadOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Thread           *models.Thread `json:"thread"`
}

var createTeamThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateTeamThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createTeamThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createTeamThreadOutput)
		return ok
	},
})

var createTeamThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(createTeamThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createTeamThreadInputType)},
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
		uuid, _ := input["uuid"].(string)
		orgID := input["organizationID"].(string)
		title, _ := input["title"].(string)
		mems, _ := input["memberEntityIDs"].([]interface{})
		members := make([]string, len(mems))
		for i, m := range mems {
			members[i] = m.(string)
		}

		creatorEnt, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if creatorEnt == nil {
			return nil, errors.ErrNotAuthorized(ctx, orgID)
		}

		thread, err := ram.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
			UUID:            uuid,
			OrganizationID:  orgID,
			FromEntityID:    creatorEnt.ID,
			Summary:         "New conversation", // TODO: not sure what we want here
			UserTitle:       title,
			MemberEntityIDs: dedupeStrings(append(members, creatorEnt.ID)),
			Type:            threading.ThreadType_TEAM,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		th, err := transformThreadToResponse(thread)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &createTeamThreadOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Thread:           th,
		}, nil
	},
}
