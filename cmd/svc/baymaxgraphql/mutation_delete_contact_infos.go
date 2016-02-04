package main

import (
	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
)

type deleteContactInfosOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var deleteContactInfosInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DeleteContactInfosInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"contactIDs":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.String)},
	},
})

var deleteContactInfosOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DeleteContactInfosPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*deleteContactInfosOutput)
		return ok
	},
})

var deleteContactInfosMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteContactInfosOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteContactInfosInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		contactIDs, _ := input["contactIDs"].([]interface{})
		entID, _ := input["entityID"].(string)

		sContacts := make([]string, len(contactIDs))
		for i, ci := range contactIDs {
			sContacts[i] = ci.(string)
		}

		resp, err := svc.directory.DeleteContacts(ctx, &directory.DeleteContactsRequest{
			EntityID:         entID,
			EntityContactIDs: sContacts,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
		})
		if err != nil {
			return nil, internalError(err)
		}

		e, err := transformEntityToResponse(resp.Entity)
		if err != nil {
			return nil, internalError(err)
		}

		return &deleteContactInfosOutput{
			ClientMutationID: mutationID,
			Entity:           e,
		}, nil
	},
}
