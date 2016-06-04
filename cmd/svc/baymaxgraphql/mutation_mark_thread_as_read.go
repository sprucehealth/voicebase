package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

/// markThreadAsRead (DEPRECATED)

type markThreadAsReadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var markThreadAsReadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MarkThreadAsReadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"threadID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var markThreadAsReadErrorCodeEnum = graphql.String

var markThreadAsReadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MarkThreadAsReadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: markThreadAsReadErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*markThreadAsReadOutput)
			return ok
		},
	},
)

var markThreadAsReadMutation = &graphql.Field{
	Type:              graphql.NewNonNull(markThreadAsReadOutputType),
	DeprecationReason: "Deprecated in favor of markThreadsAsRead mutation",
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(markThreadAsReadInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		threadID, _ := input["threadID"].(string)
		orgID, _ := input["organizationID"].(string)

		ent, err := entityInOrgForAccountID(ctx, ram, orgID, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		if err = ram.MarkThreadAsRead(ctx, threadID, ent.ID); err != nil {
			return nil, err
		}

		return &markThreadAsReadOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	}),
}

// markThreadsAsRead

type markThreadsAsReadOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var markThreadsAsReadErrorCodeEnum = graphql.String

var markThreadsAsReadOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "MarkThreadsAsReadPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: markThreadsAsReadErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*markThreadsAsReadOutput)
			return ok
		},
	},
)

type markThreadsAsReadInput struct {
	ThreadIDs        []string `gql:"threadIDs"`
	OrganizationID   string   `gql:"organizationID,nonempty"`
	ClientMutationID string   `gql:"clientMutationId"`
	AllThreads       bool     `gql:"all"`
}

var markThreadsAsReadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MarkThreadsAsReadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"threadIDs":        &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.ID)},
			"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"all":              &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

var markThreadsAsReadMutation = &graphql.Field{
	Type: graphql.NewNonNull(markThreadsAsReadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(markThreadsAsReadInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		var in markThreadsAsReadInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		// TODO: All the work

		return &markThreadsAsReadOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
		}, nil
	}),
}
