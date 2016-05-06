package main

import (
	"fmt"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
)

// callEntity

const (
	callEntityTypeConnectParties    = "CONNECT_PARTIES"
	callEntityTypeReturnPhoneNumber = "RETURN_PHONE_NUMBER"
)

const (
	callEntityErrorCodeEntityNotFound     = "ENTITY_NOT_FOUND"
	callEntityErrorCodeEntityHasNoContact = "ENTITY_HAS_NO_CONTACT"
	callEntityInvalidPhoneNumber          = "INVALID_PHONE_NUMBER"
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
		callEntityInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       callEntityInvalidPhoneNumber,
			Description: "Invalid phone number",
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
		"clientMutationId": newClientmutationIDOutputField(),
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

var callEntityMutation = &graphql.Field{
	Type: graphql.NewNonNull(callEntityOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(callEntityInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		svc := serviceFromParams(p)
		headers := gqlctx.SpruceHeaders(ctx)

		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		originatingPhoneNumber, _ := input["originatingPhoneNumber"].(string)
		destinationPhoneNumber, _ := input["destinationPhoneNumber"].(string)
		entityID := input["destinationEntityID"].(string)

		if destinationPhoneNumber == "" {
			return nil, fmt.Errorf("destination phone number for entity required")
		}

		// the phone number specified has to be one of the contact values for the external
		// entity
		pn, err := phone.ParseNumber(destinationPhoneNumber)
		if err != nil {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        callEntityInvalidPhoneNumber,
				ErrorMessage:     "The destination phone number is not a valid US phone number",
			}, nil
		} else if !pn.IsCallable() {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        callEntityInvalidPhoneNumber,
				ErrorMessage:     "The destination phone number cannot be called given that it represents an unavailable phone number.",
			}, nil
		}

		if originatingPhoneNumber != "" {
			_, err := phone.ParseNumber(originatingPhoneNumber)
			if err != nil {
				return &callEntityOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        callEntityInvalidPhoneNumber,
					ErrorMessage:     "The originating phone number is not a valid US phone number",
				}, nil
			}
		}

		calleeEnt, err := ram.Entity(ctx, entityID, []directory.EntityInformation{
			directory.EntityInformation_CONTACTS,
			directory.EntityInformation_MEMBERSHIPS,
		}, 0)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if calleeEnt == nil || (calleeEnt.Type != directory.EntityType_EXTERNAL && calleeEnt.Type != directory.EntityType_PATIENT) {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The callee was not found.",
			}, nil
		}

		numberFound := false
		for _, contact := range calleeEnt.Contacts {
			if contact.Value == pn.String() {
				numberFound = true
				break
			}
		}
		if !numberFound {
			return nil, fmt.Errorf("phone number specified is not one of entity's contact values")
		}

		var org *directory.Entity
		for _, em := range calleeEnt.Memberships {
			if em.Type == directory.EntityType_ORGANIZATION {
				org = em
				break
			}
		}
		if org == nil {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The callee was not found.",
			}, nil
		}

		callerEnt, err := ram.EntityForAccountID(ctx, org.ID, acc.ID)
		if err != nil {
			return nil, err
		}
		if callerEnt == nil {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				ErrorCode:        callEntityErrorCodeEntityNotFound,
				ErrorMessage:     "The caller was not found.",
			}, nil
		}

		ireq := &excomms.InitiatePhoneCallRequest{
			FromPhoneNumber: originatingPhoneNumber,
			ToPhoneNumber:   destinationPhoneNumber,
			CallerEntityID:  callerEnt.ID,
			OrganizationID:  org.ID,
			DeviceID:        headers.DeviceID,
		}
		switch input["type"].(string) {
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
			golog.Errorf("Unable to format proxy phone number %s :%s ", ires.ProxyPhoneNumber, err.Error())
		}

		originatingPhoneNumberDisplayValue, err := phone.Format(ires.OriginatingPhoneNumber, phone.Pretty)
		if err != nil {
			golog.Errorf("Unable to format originating phone number %s: %s", ires.OriginatingPhoneNumber, err.Error())
		}

		conc.Go(func() {
			svc.segmentio.Track(&analytics.Track{
				Event:  "outbound-call-attempted",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"org_id":                   org.ID,
					"originating_phone_number": originatingPhoneNumber,
					"proxy_phone_number":       ires.ProxyPhoneNumber,
					"platform":                 headers.Platform.String(),
				},
			})
		})

		return &callEntityOutput{
			ClientMutationID:                   mutationID,
			Success:                            true,
			ProxyPhoneNumber:                   ires.ProxyPhoneNumber,
			ProxyPhoneNumberDisplayValue:       proxyPhoneNumberDisplayValue,
			OriginatingPhoneNumberDisplayValue: originatingPhoneNumberDisplayValue,
			OriginatingPhoneNumber:             ires.OriginatingPhoneNumber,
		}, nil
	},
}
