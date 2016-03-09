package main

import (
	"fmt"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type inviteColleaguesOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var inviteColleaguesInfoType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InviteColleaguesInfo",
	Fields: graphql.InputObjectConfigFieldMap{
		"email":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"phoneNumber": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

var inviteColleaguesInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "InviteColleaguesInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"colleagues":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(inviteColleaguesInfoType)))},
	},
})

const (
	inviteColleaguesErrorCodeInvalidEmail       = "INVALID_EMAIL"
	inviteColleaguesErrorCodeInvalidPhoneNumber = "INVALID_PHONE_NUMBER"
)

var inviteColleaguesErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "InviteColleaguesErrorCode",
	Values: graphql.EnumValueConfigMap{
		inviteColleaguesErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       inviteColleaguesErrorCodeInvalidEmail,
			Description: "The provided email address is invalid",
		},
		inviteColleaguesErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       inviteColleaguesErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
	},
})

var inviteColleaguesOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "InviteColleaguesPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: inviteColleaguesErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*inviteColleaguesOutput)
		return ok
	},
})

var inviteColleaguesMutation = &graphql.Field{
	Description: "inviteColleagues invites one or more people to an organization",
	Type:        graphql.NewNonNull(inviteColleaguesOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(inviteColleaguesInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		orgID := input["organizationID"].(string)
		colleaguesInput := input["colleagues"].([]interface{})
		colleagues := make([]*invite.Colleague, len(colleaguesInput))
		for i, c := range colleaguesInput {
			m := c.(map[string]interface{})
			col := &invite.Colleague{
				Email:       m["email"].(string),
				PhoneNumber: m["phoneNumber"].(string),
			}
			if !validate.Email(col.Email) {
				return &inviteColleaguesOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        inviteColleaguesErrorCodeInvalidEmail,
					ErrorMessage:     fmt.Sprintf("The email address '%s' not valid.", col.Email),
				}, nil
			}
			var err error
			col.PhoneNumber, err = phone.Format(col.PhoneNumber, phone.E164)
			if err != nil {
				return &inviteColleaguesOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        inviteColleaguesErrorCodeInvalidEmail,
					ErrorMessage:     fmt.Sprintf("The phone number '%s' not valid.", col.PhoneNumber),
				}, nil
			}
			colleagues[i] = col
		}

		ent, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if ent == nil {
			return nil, errors.New("Not a member of the organization")
		}

		if _, err := svc.invite.InviteColleagues(ctx, &invite.InviteColleaguesRequest{
			OrganizationEntityID: orgID,
			InviterEntityID:      ent.ID,
			Colleagues:           colleagues,
		}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		for _, c := range colleagues {
			svc.segmentio.Track(&analytics.Track{
				Event:  "invited-colleague",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"email":        c.Email,
					"phone_number": c.PhoneNumber,
				},
			})
		}

		return &inviteColleaguesOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}

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
				ErrorMessage:     "The invite token does not match a valid invite.",
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
		return &associateInviteOutput{ClientMutationID: mutationID, Success: true, Values: values}, nil
	},
}
