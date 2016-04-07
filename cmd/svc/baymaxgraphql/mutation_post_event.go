package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var postEventInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "PostEventInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"eventName":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"attributes":       &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(postEventAttributeInputType))},
	},
})

var postEventAttributeInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "PostEventAttributeInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"key":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"value": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var postEventErrorCodeEnum = graphql.String

var postEventOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "PostEventPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: postEventErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*postEventOutput)
		return ok
	},
})

type postEventOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var postEventMutation = &graphql.Field{
	Type: graphql.NewNonNull(postEventOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(postEventInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		eventName := input["eventName"].(string)
		attrsIn, _ := input["attributes"].([]interface{})
		attrs := make(map[string]string, len(attrsIn))
		for _, aIn := range attrsIn {
			att, _ := aIn.(map[string]interface{})
			key, _ := att["key"].(string)
			value, _ := att["value"].(string)
			if key != "" && value != "" {
				attrs[strings.ToLower(key)] = value
			}
		}

		if strings.HasPrefix(eventName, "setup_") {
			acc := gqlctx.Account(ctx)
			if acc == nil || acc.Type != auth.AccountType_PROVIDER {
				return nil, errors.ErrNotAuthenticated(ctx)
			}

			orgID := attrs["org_id"]
			var ent *directory.Entity
			if orgID == "" {
				// TODO: for now support the event not including the orgID, eventually this shouldn't be necessary
				entities, err := ram.EntitiesForExternalID(ctx, acc.ID,
					[]directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS}, 0, []directory.EntityStatus{directory.EntityStatus_ACTIVE})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				for _, e := range entities {
					for _, em := range e.Memberships {
						if em.Type == directory.EntityType_ORGANIZATION {
							if ent != nil {
								return nil, errors.InternalError(ctx, fmt.Errorf("Expected 1 org for account %s but found more", acc.ID))
							}
							ent = e
							orgID = em.ID
						}
					}
				}
			} else {
				e, err := ram.EntityForAccountID(ctx, orgID, acc.ID)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				ent = e
			}
			if ent == nil {
				golog.Warningf("No entity found for account %s", acc.ID)
				// Nothing to do but no real reason to return an error
				return &postEventOutput{
					ClientMutationID: mutationID,
					Success:          true,
				}, nil
			}

			eventAttr := make([]*threading.KeyValue, 0, len(attrsIn))
			for k, v := range attrs {
				eventAttr = append(eventAttr, &threading.KeyValue{Key: k, Value: v})
			}
			_, err := ram.OnboardingThreadEvent(ctx, &threading.OnboardingThreadEventRequest{
				LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
				LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
					EntityID: orgID,
				},
				EventType: threading.OnboardingThreadEventRequest_GENERIC_SETUP,
				Event: &threading.OnboardingThreadEventRequest_GenericSetup{
					GenericSetup: &threading.GenericSetupEvent{
						Name:       eventName,
						Attributes: eventAttr,
					},
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		return &postEventOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
