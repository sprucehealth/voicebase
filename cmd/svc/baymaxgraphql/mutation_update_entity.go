package main

import (
	"fmt"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

type updateEntityOutput struct {
	ClientMutationID string  `json:"clientMutationId,omitempty"`
	Success          bool    `json:"success"`
	ErrorCode        string  `json:"errorCode,omitempty"`
	ErrorMessage     string  `json:"errorMessage,omitempty"`
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

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var updateEntityErrorCodeEnum = graphql.String

var updateEntityOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateEntityPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateEntityErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
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
			return nil, errNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		entID := input["entityID"].(string)
		entityInfoInput := input["entityInfo"].(map[string]interface{})
		entityInfo, err := entityInfoFromInput(entityInfoInput)
		if err != nil {
			return nil, internalError(ctx, err)
		}
		contactFields, _ := entityInfoInput["contactInfos"].([]interface{})
		contacts, err := contactListFromInput(contactFields, false)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		entityInfo.DisplayName, err = buildDisplayName(entityInfo, contacts)
		if err != nil {
			return nil, internalError(ctx, err)
		}

		serializedContactInput, _ := entityInfoInput["serializedContacts"].([]interface{})
		serializedContacts := make([]*directory.SerializedClientEntityContact, len(serializedContactInput))
		for i, sci := range serializedContactInput {
			msci := sci.(map[string]interface{})
			platform := msci["platform"].(string)
			contact := msci["contact"].(string)
			pPlatform, ok := directory.Platform_value[platform]
			if !ok {
				return nil, fmt.Errorf("Unknown platform type %s", platform)
			}
			dPlatform := directory.Platform(pPlatform)
			serializedContacts[i] = &directory.SerializedClientEntityContact{
				EntityID:                entID,
				Platform:                dPlatform,
				SerializedEntityContact: []byte(contact),
			}
		}

		resp, err := svc.directory.UpdateEntity(ctx, &directory.UpdateEntityRequest{
			EntityID:                 entID,
			EntityInfo:               entityInfo,
			Contacts:                 contacts,
			SerializedEntityContacts: serializedContacts,
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

		return &updateEntityOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
		}, nil
	},
}
