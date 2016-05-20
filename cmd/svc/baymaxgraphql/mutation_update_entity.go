package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

type updateEntityOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
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
		"clientMutationId": newClientMutationIDOutputField(),
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
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		svc := serviceFromParams(p)
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		entID := input["entityID"].(string)
		entityInfoInput := input["entityInfo"].(map[string]interface{})
		entityInfo, err := entityInfoFromInput(entityInfoInput)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		contactFields, _ := entityInfoInput["contactInfos"].([]interface{})
		contacts, err := contactListFromInput(contactFields, false)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
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

		entity, err := ram.UpdateEntity(ctx, &directory.UpdateEntityRequest{
			EntityID:                       entID,
			UpdateEntityInfo:               true,
			EntityInfo:                     entityInfo,
			UpdateContacts:                 true,
			Contacts:                       contacts,
			UpdateSerializedEntityContacts: true,
			SerializedEntityContacts:       serializedContacts,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		// update the system title for all threads that have this entity as their primary entity
		threads, err := ram.ThreadsForMember(ctx, entity.ID, true)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		for _, thread := range threads {
			if _, err := ram.UpdateThread(ctx, &threading.UpdateThreadRequest{
				ThreadID:    thread.ID,
				SystemTitle: entity.Info.DisplayName,
			}); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		sh := devicectx.SpruceHeaders(ctx)
		e, err := transformEntityToResponse(svc.staticURLPrefix, entity, sh, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &updateEntityOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
		}, nil
	},
}
