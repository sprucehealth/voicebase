package main

import (
	"github.com/sprucehealth/graphql"
)

var postEventInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "PostEventInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"eventName":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
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
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		eventName := input["eventName"].(string)

		// TODO: stubbing this mutation for now to allow apps to call it withouts errors. will fill in the blanks later
		_ = eventName

		return &postEventOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
