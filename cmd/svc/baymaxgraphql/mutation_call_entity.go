package main

import (
	"fmt"

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
	callEntityResultSuccess            = "SUCCESS"
	callEntityResultEntityNotFound     = "ENTITY_NOT_FOUND"
	callEntityResultEntityHasNoContact = "ENTITY_HAS_NO_CONTACT"
)

type callEntityOutput struct {
	ClientMutationID       string `json:"clientMutationId,omitempty"`
	Result                 string `json:"result"`
	ProxyPhoneNumber       string `json:"proxyPhoneNumber,omitempty"`
	OriginatingPhoneNumber string `json:"originatingPhoneNumber,omitempty"`
}

var callEntityTypeEnumType = graphql.NewEnum(
	graphql.EnumConfig{
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
	},
)

var callEntityResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CallEntityResult",
		Description: "Result of callEntity",
		Values: graphql.EnumValueConfigMap{
			callEntityResultSuccess: &graphql.EnumValueConfig{
				Value:       callEntityResultSuccess,
				Description: "Success",
			},
			callEntityResultEntityNotFound: &graphql.EnumValueConfig{
				Value:       callEntityResultEntityNotFound,
				Description: "The requested entity does not exist",
			},
			callEntityResultEntityHasNoContact: &graphql.EnumValueConfig{
				Value:       callEntityResultEntityHasNoContact,
				Description: "An entity does not have a viable contact",
			},
		},
	},
)

var callEntityInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
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
	},
)

var callEntityOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CallEntityPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(callEntityResultType)},
			"proxyPhoneNumber": &graphql.Field{
				Type:        graphql.String,
				Description: "The phone number to use to contact the entity.",
			},
			"originatingPhoneNumber": &graphql.Field{
				Type:        graphql.String,
				Description: "The phone number of where the call is intended to originate from",
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*callEntityOutput)
			return ok
		},
	},
)

var callEntityMutation = &graphql.Field{
	Type: graphql.NewNonNull(callEntityOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(callEntityInputType)},
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
		originatingPhoneNumber, _ := input["originatingPhoneNumber"].(string)
		destinationPhoneNumber, _ := input["destinationPhoneNumber"].(string)
		entityID := input["destinationEntityID"].(string)

		if destinationPhoneNumber == "" {
			return nil, fmt.Errorf("destination phone number for entity required")
		}

		calleeEnt, err := svc.entity(ctx, entityID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if calleeEnt == nil || calleeEnt.Type != directory.EntityType_EXTERNAL {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				Result:           callEntityResultEntityNotFound,
			}, nil
		}

		// the phone number specified has to be one of the contact values for the external
		// entity
		pn, err := phone.ParseNumber(destinationPhoneNumber)
		if err != nil {
			return nil, fmt.Errorf("invalid format for US phone number: %s", err.Error())
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
				Result:           callEntityResultEntityNotFound,
			}, nil
		}

		callerEnt, err := svc.entityForAccountID(ctx, org.ID, acc.ID)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		if callerEnt == nil {
			return &callEntityOutput{
				ClientMutationID: mutationID,
				Result:           callEntityResultEntityNotFound,
			}, nil
		}

		ireq := &excomms.InitiatePhoneCallRequest{
			FromPhoneNumber: originatingPhoneNumber,
			ToPhoneNumber:   destinationPhoneNumber,
			CallerEntityID:  callerEnt.ID,
			OrganizationID:  org.ID,
		}
		switch input["type"].(string) {
		case callEntityTypeConnectParties:
			ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_CONNECT_PARTIES
		case callEntityTypeReturnPhoneNumber:
			ireq.CallInitiationType = excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER
		}
		ires, err := svc.exComms.InitiatePhoneCall(ctx, ireq)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		return &callEntityOutput{
			ClientMutationID:       mutationID,
			Result:                 callEntityResultSuccess,
			ProxyPhoneNumber:       ires.ProxyPhoneNumber,
			OriginatingPhoneNumber: ires.OriginatingPhoneNumber,
		}, nil
	},
}
