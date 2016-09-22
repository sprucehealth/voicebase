package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// updateSavedThreadQuery
type updateSavedThreadQueryInput struct {
	ClientMutationID     string `gql:"clientMutationId"`
	SavedQueryID         string `gql:"savedQueryID,nonempty"`
	NotificationsEnabled bool   `gql:"notificationsEnabled,nonempty"`
}

var updateSavedThreadQueryInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateSavedThreadQueryInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId":     newClientMutationIDInputField(),
			"savedQueryID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"notificationsEnabled": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

const updateSavedThreadQueryErrorCode = "UpdateSavedThreadQueryErrorCode"

var updateSavedThreadQueryErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "UpdateSavedThreadQueryErrorCode",
	Values: graphql.EnumValueConfigMap{
		updateSavedThreadQueryErrorCode: &graphql.EnumValueConfig{
			Value:       updateSavedThreadQueryErrorCode,
			Description: "Placeholder",
		},
	},
})

type updateSavedThreadQueryOutput struct {
	ClientMutationID string                   `json:"clientMutationId,omitempty"`
	Success          bool                     `json:"success"`
	ErrorCode        string                   `json:"errorCode,omitempty"`
	ErrorMessage     string                   `json:"errorMessage,omitempty"`
	SavedThreadQuery *models.SavedThreadQuery `json:"savedThreadQuery"`
}

var updateSavedThreadQueryOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateSavedThreadQueryPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: updateSavedThreadQueryErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"savedThreadQuery": &graphql.Field{Type: savedThreadQueryType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateSavedThreadQueryOutput)
			return ok
		},
	},
)

var updateSavedThreadQueryMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateSavedThreadQueryOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateSavedThreadQueryInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in updateSavedThreadQueryInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return updateSavedThreadQuery(p, in)
	}),
}

func updateSavedThreadQuery(p graphql.ResolveParams, in updateSavedThreadQueryInput) (interface{}, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)

	uReq := &threading.UpdateSavedQueryRequest{
		SavedQueryID:         in.SavedQueryID,
		NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_NONE,
	}
	if in.NotificationsEnabled {
		uReq.NotificationsEnabled = threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE
	} else {
		uReq.NotificationsEnabled = threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE
	}

	resp, err := ram.UpdateSavedQuery(ctx, uReq)
	if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}

	sqResp, err := transformSavedQueryToResponse(resp.Query)
	if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}

	return &updateSavedThreadQueryOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		SavedThreadQuery: sqResp,
	}, nil
}
