package main

import (
	"fmt"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
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
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
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

		ent, err := svc.entityForAccountID(ctx, orgID, acc.ID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if ent == nil {
			return nil, errors.New("Not a member of the organization")
		}

		if _, err := svc.invite.InviteColleagues(ctx, &invite.InviteColleaguesRequest{
			OrganizationEntityID: orgID,
			InviterEntityID:      ent.ID,
			Colleagues:           colleagues,
		}); err != nil {
			return nil, internalError(ctx, err)
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
		sh := spruceHeadersFromContext(ctx)
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
			return nil, internalError(ctx, err)
		}

		return &associateAttributionOutput{ClientMutationID: mutationID, Success: true}, nil
	},
}
