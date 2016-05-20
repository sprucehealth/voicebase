package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

type deleteContactInfosOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Entity           *models.Entity `json:"entity"`
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

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var deleteContactInfosErrorCodeEnum = graphql.String

var deleteContactInfosOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DeleteContactInfosPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: deleteContactInfosErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
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
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		svc := serviceFromParams(p)
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		contactIDs, _ := input["contactIDs"].([]interface{})
		entID := input["entityID"].(string)

		sContacts := make([]string, len(contactIDs))
		for i, ci := range contactIDs {
			sContacts[i] = ci.(string)
		}

		ent, err := ram.DeleteContacts(ctx, &directory.DeleteContactsRequest{
			EntityID:         entID,
			EntityContactIDs: sContacts,
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		sh := devicectx.SpruceHeaders(ctx)
		e, err := transformEntityToResponse(svc.staticURLPrefix, ent, sh, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &deleteContactInfosOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
		}, nil
	},
}
