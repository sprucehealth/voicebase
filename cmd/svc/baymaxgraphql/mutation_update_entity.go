package main

import (
	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/svc/directory"
)

type updateEntityOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Entity           *entity `json:"entity"`
}

var updateEntityInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateEntityInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"entityInfo":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(entityInfoInputType)},
	},
})

var updateEntityOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateEntityPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"entity":           &graphql.Field{Type: graphql.NewNonNull(entityType)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*updateEntityOutput)
		return ok
	},
})

var updateEntityMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateEntityOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateEntityInputType)},
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
		entID := input["entityID"].(string)
		entityInfoInput := input["entityInfo"].(map[string]interface{})
		entityInfo, err := entityInfoFromInput(entityInfoInput)
		if err != nil {
			return nil, internalError(err)
		}
		contactFields, _ := entityInfoInput["contactInfos"].([]interface{})
		contacts, err := contactListFromInput(contactFields, false)
		if err != nil {
			return nil, internalError(err)
		}

		entityInfo.DisplayName, err = buildDisplayName(entityInfo, contacts)
		if err != nil {
			return nil, internalError(err)
		}

		resp, err := svc.directory.UpdateEntity(ctx, &directory.UpdateEntityRequest{
			EntityID:   entID,
			EntityInfo: entityInfo,
			Contacts:   contacts,
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

		return &updateEntityOutput{
			ClientMutationID: mutationID,
			Entity:           e,
		}, nil
	},
}
