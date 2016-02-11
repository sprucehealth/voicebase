package main

import (
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

type updateContactInfosOutput struct {
	ClientMutationID string  `json:"clientMutationId,omitempty"`
	Success          bool    `json:"success"`
	ErrorCode        string  `json:"errorCode,omitempty"`
	ErrorMessage     string  `json:"errorMessage,omitempty"`
	Entity           *entity `json:"entity"`
}

var updateContactInfosInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateContactInfosInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"contactInfos":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(contactInfoInputType)},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var updateContactInfosErrorCodeEnum = graphql.String

var updateContactInfosOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateContactInfosPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateContactInfosErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*updateContactInfosOutput)
		return ok
	},
})

var updateContactInfosMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateContactInfosOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateContactInfosInputType)},
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
		contactInfos, _ := input["contactInfos"].([]interface{})
		entID := input["entityID"].(string)

		contacts, err := contactListFromInput(contactInfos, false)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		resp, err := svc.directory.UpdateContacts(ctx, &directory.UpdateContactsRequest{
			EntityID: entID,
			Contacts: contacts,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
		})
		if err != nil {
			return nil, internalError(ctx, err)
		}

		e, err := transformEntityToResponse(resp.Entity)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		return &updateContactInfosOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
		}, nil
	},
}
