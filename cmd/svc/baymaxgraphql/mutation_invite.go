package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// associateAttribution

type associateAttributionOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var associateAttributionValueType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "AssociateAttributionValue",
	Fields: graphql.InputObjectConfigFieldMap{
		"key":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var associateAttributionInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AssociateAttributionInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"values":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(associateAttributionValueType)))},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var associateAttributionErrorCodeEnum = graphql.String

var associateAttributionOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AssociateAttributionPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: associateAttributionErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*associateAttributionOutput)
			return ok
		},
	},
)

var associateAttributionMutation = &graphql.Field{
	Description: "associateAttribution attaches attribution information to the device ID of the requester",
	Type:        graphql.NewNonNull(associateAttributionOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(associateAttributionInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		sh := gqlctx.SpruceHeaders(ctx)
		if sh.DeviceID == "" {
			return nil, errors.New("missing device ID")
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		valuesInput := input["values"].([]interface{})
		values := make([]*invite.AttributionValue, len(valuesInput))
		for i, v := range valuesInput {
			m := v.(map[string]interface{})
			value, _ := m["value"].(string)
			if value != "" {
				values[i] = &invite.AttributionValue{
					Key:   m["key"].(string),
					Value: value,
				}
			}
		}
		_, err := svc.invite.SetAttributionData(ctx, &invite.SetAttributionDataRequest{
			DeviceID: sh.DeviceID,
			Values:   values,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &associateAttributionOutput{ClientMutationID: mutationID, Success: true}, nil
	},
}

// associateInvite

var inviteValueType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InviteValue",
	Fields: graphql.Fields{
		"key":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

type inviteValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type associateInviteOutput struct {
	ClientMutationID string        `json:"clientMutationId,omitempty"`
	Success          bool          `json:"success"`
	ErrorCode        string        `json:"errorCode,omitempty"`
	ErrorMessage     string        `json:"errorMessage,omitempty"`
	InviteType       string        `json:"inviteType"`
	Values           []inviteValue `json:"values,omitempty"`
}

var associateInviteValueType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "AssociateInviteValue",
	Fields: graphql.InputObjectConfigFieldMap{
		"key":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var associateInviteInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AssociateInviteInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

const associateInviteErrorCodeInvalidInvite = "INVALID_INVITE"

var associateInviteErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AssociateInviteErrorCode",
	Values: graphql.EnumValueConfigMap{
		associateInviteErrorCodeInvalidInvite: &graphql.EnumValueConfig{
			Value:       associateInviteErrorCodeInvalidInvite,
			Description: "The provided token doesn't match a valid invite",
		},
	},
})

var associateInviteOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AssociateInvitePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: associateInviteErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"inviteType":       &graphql.Field{Type: inviteTypeEnum},
			"values": &graphql.Field{
				Type:        graphql.NewList(graphql.NewNonNull(inviteValueType)),
				Description: "Values is the set of data attached to the invite which matters the attribution data from Branch",
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*associateInviteOutput)
			return ok
		},
	},
)

const (
	inviteTypeUnknown   = "UNKNOWN"
	inviteTypePatient   = "PATIENT"
	inviteTypeColleague = "COLLEAGUE"
)

var inviteTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InviteType",
	Values: graphql.EnumValueConfigMap{
		inviteTypeUnknown: &graphql.EnumValueConfig{
			Value:       inviteTypeUnknown,
			Description: "Indicates that the provided invite code was mapped to an unknown type",
		},
		inviteTypePatient: &graphql.EnumValueConfig{
			Value:       inviteTypePatient,
			Description: "Indicates that the provided invite code was for a patient invite",
		},
		inviteTypeColleague: &graphql.EnumValueConfig{
			Value:       inviteTypeColleague,
			Description: "Indicates that the provided invite code was for a provider invite",
		},
	},
})

var associateInviteMutation = &graphql.Field{
	Description: "associateInvite looks up an invite by token, attaches the attribution data to the device ID, and returns the attribution data",
	Type:        graphql.NewNonNull(associateInviteOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(associateInviteInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		sh := gqlctx.SpruceHeaders(ctx)
		if sh.DeviceID == "" {
			return nil, errors.New("missing device ID")
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		res, err := svc.invite.LookupInvite(ctx, &invite.LookupInviteRequest{
			Token: token,
		})
		if grpc.Code(err) == codes.NotFound {
			return &associateInviteOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        associateInviteErrorCodeInvalidInvite,
				ErrorMessage:     "Sorry, the invite code you entered is not valid. Please re-enter the code or contact your healthcare provider.",
			}, nil
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if _, err := svc.invite.SetAttributionData(ctx, &invite.SetAttributionDataRequest{
			DeviceID: sh.DeviceID,
			Values:   res.Values,
		}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		values := make([]inviteValue, len(res.Values))
		for i, v := range res.Values {
			values[i] = inviteValue{Key: v.Key, Value: v.Value}
		}

		return &associateInviteOutput{
			ClientMutationID: mutationID,
			Success:          true,
			InviteType:       inviteTypeToEnum(res.Type),
			Values:           values,
		}, nil
	},
}

func inviteTypeToEnum(t invite.LookupInviteResponse_Type) string {
	switch t {
	case invite.LookupInviteResponse_PATIENT:
		return inviteTypePatient
	case invite.LookupInviteResponse_COLLEAGUE:
		return inviteTypeColleague
	default:
		golog.Errorf("Unknown invite type %s, returning unknown", t.String())
	}
	return inviteTypeUnknown
}
