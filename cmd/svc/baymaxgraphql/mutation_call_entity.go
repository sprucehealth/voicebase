package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// callEntity

const (
	callEntityTypeConnectParties    = "CONNECT_PARTIES"
	callEntityTypeReturnPhoneNumber = "RETURN_PHONE_NUMBER"
)

const (
	callEntityErrorCodeEntityNotFound     = "ENTITY_NOT_FOUND"
	callEntityErrorCodeEntityHasNoContact = "ENTITY_HAS_NO_CONTACT"
	callEntityErrorCodeInvalidPhoneNumber = "INVALID_PHONE_NUMBER"
)

type callEntityOutput struct {
	ClientMutationID                   string `json:"clientMutationId,omitempty"`
	Success                            bool   `json:"success"`
	ErrorCode                          string `json:"errorCode,omitempty"`
	ErrorMessage                       string `json:"errorMessage,omitempty"`
	ProxyPhoneNumber                   string `json:"proxyPhoneNumber,omitempty"`
	ProxyPhoneNumberDisplayValue       string `json:"proxyPhoneNumberDisplayValue,omitempty"`
	OriginatingPhoneNumber             string `json:"originatingPhoneNumber,omitempty"`
	OriginatingPhoneNumberDisplayValue string `json:"originatingPhoneNumberDisplayValue,omitempty"`
}

var callEntityTypeEnumType = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CallEntityType",
	Description: "How to initiate the call",
	Values: graphql.EnumValueConfigMap{
		callEntityTypeConnectParties: &graphql.EnumValueConfig{
			Value:       callEntityTypeConnectParties,
			Description: "Connect parties by calling both numbers",
		},
		callEntityTypeReturnPhoneNumber: &graphql.EnumValueConfig{
			Value:       callEntityTypeReturnPhoneNumber,
			Description: "Return a phone number to call",
		},
	},
})

var callEntityErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "CallEntityErrorCode",
	Description: "Result of callEntity",
	Values: graphql.EnumValueConfigMap{
		callEntityErrorCodeEntityNotFound: &graphql.EnumValueConfig{
			Value:       callEntityErrorCodeEntityNotFound,
			Description: "The requested entity does not exist",
		},
		callEntityErrorCodeEntityHasNoContact: &graphql.EnumValueConfig{
			Value:       callEntityErrorCodeEntityHasNoContact,
			Description: "An entity does not have a viable contact",
		},
		callEntityErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       callEntityErrorCodeInvalidPhoneNumber,
			Description: "Invalid phone number",
		},
	},
})

var callEntityInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CallEntityInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"destinationEntityID": &graphql.InputObjectFieldConfig{
			Type:        graphql.NewNonNull(graphql.ID),
			Description: "EntityID of the person being called.",
		},
		"originatingPhoneNumber": &graphql.InputObjectFieldConfig{
			Type:        graphql.String,
			Description: "Number from which the call is intended to originate. If one is not specified, it is assumed that the call will originate from the phone number associated with the callerEntity's account.",
		},
		"destinationPhoneNumber": &graphql.InputObjectFieldConfig{
			Type:        graphql.NewNonNull(graphql.String),
			Description: "Phone number to call. This has to map to one of the phone numbers associated with the calleeEntity.",
		},
		"type": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(callEntityTypeEnumType)},
	},
})

var callEntityOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CallEntityPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: callEntityErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"proxyPhoneNumber": &graphql.Field{
			Type:        graphql.String,
			Description: "The phone number to use to contact the entity.",
		},
		"proxyPhoneNumberDisplayValue": &graphql.Field{
			Type:        graphql.String,
			Description: "Display ready proxy phone number",
		},
		"originatingPhoneNumber": &graphql.Field{
			Type:        graphql.String,
			Description: "The phone number of where the call is intended to originate from",
		},
		"originatingPhoneNumberDisplayValue": &graphql.Field{
			Type:        graphql.String,
			Description: "Display ready originating phone number",
		},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*callEntityOutput)
		return ok
	},
})

type callEntityInput struct {
	ClientMutationID       string `gql:"clientMutationId"`
	DestinationEntityID    string `gql:"destinationEntityID"`
	OriginatingPhoneNumber string `gql:"originatingPhoneNumber"`
	DestinationPhoneNumber string `gql:"destinationPhoneNumber"`
	Type                   string `gql:"type"`
}

var callEntityMutation = &graphql.Field{
	Type: graphql.NewNonNull(callEntityOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(callEntityInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		headers := devicectx.SpruceHeaders(ctx)

		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		var in callEntityInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		if in.DestinationPhoneNumber == "" {
			return nil, fmt.Errorf("destination phone number for entity required")
		}

		// Validate the originating and destination phone numbers

		destinationNumber, err := phone.ParseNumber(in.DestinationPhoneNumber)
		if err != nil {
			return &callEntityOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        callEntityErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "The destination phone number is not a valid US phone number",
			}, nil
		} else if !destinationNumber.IsCallable() {
			return &callEntityOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        callEntityErrorCodeInvalidPhoneNumber,
				ErrorMessage:     "The destination phone number cannot be called given that it represents an unavailable phone number.",
			}, nil
		}

		if in.OriginatingPhoneNumber != "" {
			_, err := phone.ParseNumber(in.OriginatingPhoneNumber)
			if err != nil {
				return &callEntityOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        callEntityErrorCodeInvalidPhoneNumber,
					ErrorMessage:     "The originating phone number is not a valid US phone number",
				}, nil
			}
		}

		// Lookup the callee entity and make sure the destination phone number is in the contacts

		calleeEnt, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: in.DestinationEntityID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
			},
			Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if calleeEnt == nil {
			return &callEntityOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The callee was not found.",
			}, nil
		}

		numberFound := false
		for _, contact := range calleeEnt.Contacts {
			if contact.Value == destinationNumber.String() {
				numberFound = true
				break
			}
		}
		if !numberFound {
			return nil, fmt.Errorf("phone number specified is not one of entity's contact values")
		}

		// Lookup the caller in the same org as the callee

		var org *directory.Entity
		for _, em := range calleeEnt.Memberships {
			if em.Type == directory.EntityType_ORGANIZATION {
				org = em
				break
			}
		}
		if org == nil {
			return &callEntityOutput{
				ClientMutationID: in.ClientMutationID,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The callee was not found.",
			}, nil
		}

		callerEnt, err := entityInOrgForAccountID(ctx, ram, org.ID, acc)
		if err != nil {
			return nil, err
		}
		if callerEnt == nil {
			return &callEntityOutput{
				ClientMutationID: in.ClientMutationID,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The caller was not found.",
			}, nil
		}

		ireq := &excomms.InitiatePhoneCallRequest{
			FromPhoneNumber: in.OriginatingPhoneNumber,
			ToPhoneNumber:   in.DestinationPhoneNumber,
			CallerEntityID:  callerEnt.ID,
			OrganizationID:  org.ID,
			DeviceID:        headers.DeviceID,
		}
		switch in.Type {
		case callEntityTypeConnectParties:
			ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_CONNECT_PARTIES
		case callEntityTypeReturnPhoneNumber:
			ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER
		}
		ires, err := ram.InitiatePhoneCall(ctx, ireq)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		proxyPhoneNumberDisplayValue, err := phone.Format(ires.ProxyPhoneNumber, phone.Pretty)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Unable to format proxy phone number %s: %s", ires.ProxyPhoneNumber, err)
		}

		originatingPhoneNumberDisplayValue, err := phone.Format(ires.OriginatingPhoneNumber, phone.Pretty)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Unable to format originating phone number %s: %s", ires.OriginatingPhoneNumber, err)
		}

		analytics.SegmentTrack(&segment.Track{
			Event:  "outbound-call-attempted",
			UserId: acc.ID,
			Properties: map[string]interface{}{
				"org_id":                   org.ID,
				"originating_phone_number": in.OriginatingPhoneNumber,
				"proxy_phone_number":       ires.ProxyPhoneNumber,
				"platform":                 headers.Platform.String(),
			},
		})

		return &callEntityOutput{
			ClientMutationID:                   in.ClientMutationID,
			Success:                            true,
			ProxyPhoneNumber:                   ires.ProxyPhoneNumber,
			ProxyPhoneNumberDisplayValue:       proxyPhoneNumberDisplayValue,
			OriginatingPhoneNumberDisplayValue: originatingPhoneNumberDisplayValue,
			OriginatingPhoneNumber:             ires.OriginatingPhoneNumber,
		}, nil
	},
}
