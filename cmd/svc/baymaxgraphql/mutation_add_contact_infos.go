package main

import (
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

type addContactInfosOutput struct {
	ClientMutationID string  `json:"clientMutationId,omitempty"`
	Success          bool    `json:"success"`
	ErrorCode        string  `json:"errorCode,omitempty"`
	ErrorMessage     string  `json:"errorMessage,omitempty"`
	Entity           *entity `json:"entity"`
}

var addContactInfosInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AddContactsInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"contactInfos":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(contactInfoInputType)},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var addContactInfosErrorCodeEnum = graphql.String

var addContactInfosOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AddContactInfosPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: addContactInfosErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*addContactInfosOutput)
			return ok
		},
	},
)

var addContactInfosMutation = &graphql.Field{
	Type: graphql.NewNonNull(addContactInfosOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(addContactInfosInputType)},
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
		entID, _ := input["entityID"].(string)

		contacts, err := contactListFromInput(contactInfos, false)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		resp, err := svc.directory.CreateContacts(ctx, &directory.CreateContactsRequest{
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

		return &addContactInfosOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
		}, nil
	},
}
