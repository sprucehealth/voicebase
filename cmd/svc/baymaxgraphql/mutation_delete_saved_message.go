package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

type deleteSavedMessageOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var deleteSavedMessageInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DeleteSavedMessageInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"savedMessageID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	deleteSavedMessageErrorCodePlaceholder = "PLACEHOLDER"
)

var deleteSavedMessageErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "DeleteSavedMessageErrorCode",
	Values: graphql.EnumValueConfigMap{
		deleteSavedMessageErrorCodePlaceholder: &graphql.EnumValueConfig{
			Value:       deleteSavedMessageErrorCodePlaceholder,
			Description: "Placeholder",
		},
	},
})

var deleteSavedMessageOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DeleteSavedMessagePayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: deleteSavedMessageErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*deleteSavedMessageOutput)
		return ok
	},
})

type deleteSavedMessageInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	SavedMessageID   string `gql:"savedMessageID"`
}

var deleteSavedMessageMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteSavedMessageOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteSavedMessageInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		var in deleteSavedMessageInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		// Make sure saved message exists (wasn't deleted) and get organization ID to be able to fetch entity for the account
		res, err := ram.SavedMessages(ctx, &threading.SavedMessagesRequest{
			By: &threading.SavedMessagesRequest_IDs{
				IDs: &threading.IDList{IDs: []string{in.SavedMessageID}},
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		// Idempotent so this is the same as success
		if len(res.SavedMessages) == 0 {
			return &deleteSavedMessageOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
			}, nil
		}
		sm := res.SavedMessages[0]

		// Make sure the accounts has access to the saved message
		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, sm.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, sm.ID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if sm.OwnerEntityID != sm.OrganizationID && sm.OwnerEntityID != ent.ID {
			return nil, errors.ErrNotAuthorized(ctx, sm.ID)
		}

		if _, err := ram.DeleteSavedMessage(ctx, &threading.DeleteSavedMessageRequest{SavedMessageID: sm.ID}); err != nil {
			return nil, err
		}

		return &deleteSavedMessageOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
		}, nil
	}),
}
