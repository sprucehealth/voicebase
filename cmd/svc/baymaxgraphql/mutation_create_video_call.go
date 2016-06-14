package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

const (
	createCallErrorCodeInvalidRecipient  = "INVALID_RECIPIENT"
	createCallErrorCodeCallingNotAllowed = "CALLING_NOT_ALLOWED"
)

var createCallErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CreateCallErrorCode",
	Description: "Result of createVoiceCall and createVideoCall mutations",
	Values: graphql.EnumValueConfigMap{
		createCallErrorCodeInvalidRecipient: &graphql.EnumValueConfig{
			Value:       createCallErrorCodeInvalidRecipient,
			Description: "The client attempted to call a recipient that is invalid or that it's not allowed to call",
		},
		createCallErrorCodeCallingNotAllowed: &graphql.EnumValueConfig{
			Value:       createCallErrorCodeCallingNotAllowed,
			Description: "The client is not allowed to place calls",
		},
	},
})

var createCallPayloadType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateCallPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createCallErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"call":             &graphql.Field{Type: callType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createCallPayload)
		return ok
	},
})

var createCallInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateCallInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":         newClientMutationIDInputField(),
		"uuid":                     newUUIDInputField(),
		"organizationID":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"recipientCallEndpointIDs": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.ID)))},
	},
})

type createCallInput struct {
	ClientMutationID         string   `gql:"clientMutationId"`
	UUID                     string   `gql:"uuid"`
	OrganizationID           string   `gql:"organizationID,nonempty"`
	RecipientCallEndpointIDs []string `gql:"recipientCallEndpointIDs"`
}

type createCallPayload struct {
	ClientMutationID string       `json:"clientMutationId,omitempty"`
	Success          bool         `json:"success"`
	ErrorCode        string       `json:"errorCode,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	Call             *models.Call `json:"call,omitempty"`
}

var createVideoCallMutation = &graphql.Field{
	Type: graphql.NewNonNull(createCallPayloadType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createCallInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		var in createCallInput
		if err := gqldecode.Decode(input, &in); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if acc.Type != auth.AccountType_PROVIDER {
			return &createCallPayload{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createCallErrorCodeCallingNotAllowed,
				ErrorMessage:     "Sorry, you are not allowed to make video calls.",
			}, nil
		}

		if len(in.RecipientCallEndpointIDs) != 1 {
			return &createCallPayload{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createCallErrorCodeInvalidRecipient,
				ErrorMessage:     "You may only start a video call with one person at a time.",
			}, nil
		}

		caller, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, in.OrganizationID)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if caller == nil {
			return nil, errors.ErrNotAuthorized(ctx, in.OrganizationID)
		}

		recipient, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: in.RecipientCallEndpointIDs[0],
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if recipient == nil {
			return &createCallPayload{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        createCallErrorCodeInvalidRecipient,
				ErrorMessage:     "The person you're trying to call was not found.",
			}, nil
		}

		res, err := ram.InitiateIPCall(ctx, &excomms.InitiateIPCallRequest{
			Type:               excomms.IPCallType_VIDEO,
			CallerEntityID:     caller.ID,
			RecipientEntityIDs: []string{recipient.ID},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		call, err := transformCallToResponse(res.Call)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &createCallPayload{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Call:             call,
		}, nil
	}),
}
