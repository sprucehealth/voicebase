package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/svc/settings"
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

const (
	createTeamThreadErrorCodeFeatureDisabled = "FEATURE_DISABLED"
)

var createTeamThreadErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CreateTeamThreadErrorCode",
	Description: "Result of creaeteTeamThread mutation",
	Values: graphql.EnumValueConfigMap{
		createTeamThreadErrorCodeFeatureDisabled: &graphql.EnumValueConfig{
			Value:       createTeamThreadErrorCodeFeatureDisabled,
			Description: "This feature is not enabled for the org",
		},
	},
})

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
		"clientMutationId": newClientMutationIDOutputField(),
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
		svc := serviceFromParams(p)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		uuid, _ := input["uuid"].(string)
		orgID := input["organizationID"].(string)
		title, _ := input["title"].(string)
		mems, _ := input["memberEntityIDs"].([]interface{})

		// don't allow creation of team thread if setting is disabled
		teamConversationsSettingValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
			NodeID: orgID,
			Keys: []*settings.ConfigKey{
				{
					Key: baymaxgraphqlsettings.ConfigKeyTeamConversations,
				},
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		} else if !teamConversationsSettingValue.Value {
			return &createTeamThreadOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        createTeamThreadErrorCodeFeatureDisabled,
				ErrorMessage:     "We're sorry but we cannot create this team conversation as this feature is disabled for your organization.",
			}, nil
		}

		members := make([]string, len(mems))
		for i, m := range mems {
			members[i] = m.(string)
		}

		creatorEnt, err := entityInOrgForAccountID(ctx, ram, orgID, acc)
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
			Type:            threading.THREAD_TYPE_TEAM,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		th, err := transformThreadToResponse(ctx, ram, thread, acc)
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
