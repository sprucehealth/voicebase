package main

import (
	"fmt"
	"strings"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
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
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: postEventErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*postEventOutput)
		return ok
	},
})

type postEventInput struct {
	ClientMutationID string                    `gql:"clientMutationId"`
	UUID             string                    `gql:"uuid"`
	EventName        string                    `gql:"eventName"`
	Attributes       []postEventAttributeInput `gql:"attributes"`
}

type postEventAttributeInput struct {
	Key   string `gql:"key"`
	Value string `gql:"value"`
}

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
		var in postEventInput
		if err := gqldecode.Decode(input, &in); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if strings.HasPrefix(in.EventName, "setup_") {
			acc := gqlctx.Account(ctx)
			if acc == nil || acc.Type != auth.AccountType_PROVIDER {
				return nil, errors.ErrNotAuthenticated(ctx)
			}

			var orgID string
			for _, at := range in.Attributes {
				if at.Key == "org_id" {
					orgID = at.Value
				}
			}
			var ent *directory.Entity
			if orgID == "" {
				// TODO: for now support the event not including the orgID, eventually this shouldn't be necessary
				entities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
						ExternalID: acc.ID,
					},
					RequestedInformation: &directory.RequestedInformation{
						Depth:             0,
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
				})
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
				e, err := entityInOrgForAccountID(ctx, ram, orgID, acc)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				ent = e
			}
			if ent == nil {
				golog.Warningf("No entity found for account %s", acc.ID)
				// Nothing to do but no real reason to return an error
				return &postEventOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          true,
				}, nil
			}

			segmentProps := make(map[string]interface{}, len(in.Attributes)+1)
			eventAttr := make([]*threading.KeyValue, 0, len(in.Attributes))
			for _, at := range in.Attributes {
				at.Key = strings.ToLower(strings.TrimSpace(at.Key))
				if at.Key != "" && at.Value != "" {
					eventAttr = append(eventAttr, &threading.KeyValue{Key: at.Key, Value: at.Value})
					segmentProps[at.Key] = at.Value
				}
			}
			analytics.SegmentTrack(&segment.Track{
				Event:      in.EventName,
				UserId:     acc.ID,
				Properties: segmentProps,
			})
			_, err := ram.OnboardingThreadEvent(ctx, &threading.OnboardingThreadEventRequest{
				LookupByType: threading.OnboardingThreadEventRequest_ENTITY_ID,
				LookupBy: &threading.OnboardingThreadEventRequest_EntityID{
					EntityID: orgID,
				},
				EventType: threading.OnboardingThreadEventRequest_GENERIC_SETUP,
				Event: &threading.OnboardingThreadEventRequest_GenericSetup{
					GenericSetup: &threading.GenericSetupEvent{
						Name:       in.EventName,
						Attributes: eventAttr,
					},
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		return &postEventOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
		}, nil
	},
}
