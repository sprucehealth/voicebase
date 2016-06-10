package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

const (
	updateCallErrorCodeInvalidRecipient = "INVALID_STATE_TRANSITION"
)

var updateCallErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "UpdateCallErrorCode",
	Description: "Result of updateCall mutation",
	Values: graphql.EnumValueConfigMap{
		updateCallErrorCodeInvalidRecipient: &graphql.EnumValueConfig{
			Value:       updateCallErrorCodeInvalidRecipient,
			Description: "The client attempted to modify the state of the call in a way that is not allowed or doesn't make sense",
		},
	},
})

var updateCallPayloadType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateCallPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateCallErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"call":             &graphql.Field{Type: callType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*updateCallPayload)
		return ok
	},
})

var updateCallInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateCallInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"callID":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"callState":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(callStateEnum)},
	},
})

type updateCallInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	UUID             string `gql:"uuid"`
	CallID           string `gql:"callID"`
	CallState        string `gql:"callState"`
}

type updateCallPayload struct {
	ClientMutationID string       `json:"clientMutationId,omitempty"`
	Success          bool         `json:"success"`
	ErrorCode        string       `json:"errorCode,omitempty"`
	ErrorMessage     string       `json:"errorMessage,omitempty"`
	Call             *models.Call `json:"call,omitempty"`
}

var updateCallMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateCallPayloadType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateCallInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		var in updateCallInput
		if err := gqldecode.Decode(input, &in); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var state excomms.IPCallState
		switch in.CallState {
		default:
			return nil, errors.InternalError(ctx, errors.New("Unknown state "+in.CallState))
		case models.CallStatePending:
			state = excomms.IPCallState_PENDING
		case models.CallStateAccepted:
			state = excomms.IPCallState_ACCEPTED
		case models.CallStateDeclined:
			state = excomms.IPCallState_DECLINED
		case models.CallStateConnected:
			state = excomms.IPCallState_CONNECTED
		case models.CallStateFailed:
			state = excomms.IPCallState_FAILED
		case models.CallStateCompleted:
			state = excomms.IPCallState_COMPLETED
		}

		res, err := ram.UpdateIPCall(ctx, &excomms.UpdateIPCallRequest{
			IPCallID:  in.CallID,
			AccountID: acc.ID,
			State:     state,
		})
		// TODO: handle invalid state transition errors specifically
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		call, err := transformCallToResponse(res.Call, acc.ID)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &updateCallPayload{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Call:             call,
		}, nil
	}),
}
