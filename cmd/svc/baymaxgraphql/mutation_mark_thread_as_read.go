package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
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
			"clientMutationId":   newClientMutationIDOutputField(),
			"success":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":          &graphql.Field{Type: markThreadAsReadErrorCodeEnum},
			"errorMessage":       &graphql.Field{Type: graphql.String},
			"savedThreadQueries": savedThreadQueriesField,
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

		_, err = ram.MarkThreadsAsRead(ctx, &threading.MarkThreadsAsReadRequest{
			ThreadWatermarks: []*threading.MarkThreadsAsReadRequest_ThreadWatermark{
				{
					ThreadID: threadID,
				},
			},
			EntityID: ent.ID,
			Seen:     true,
		})
		if err != nil {
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
	ClientMutationID string           `json:"clientMutationId,omitempty"`
	Success          bool             `json:"success"`
	ErrorCode        string           `json:"errorCode,omitempty"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	Threads          []*models.Thread `json:"threads"`
	threadIDs        []string
	entity           *directory.Entity
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
			"threads": &graphql.Field{
				Type: graphql.NewList(threadType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					ram := raccess.ResourceAccess(p)
					acc := gqlctx.Account(ctx)

					output := p.Source.(*markThreadsAsReadOutput)
					if len(output.threadIDs) == 0 {
						return nil, nil
					}

					// requery all the threads in the list of marking threads as read
					res, err := ram.Threads(ctx, &threading.ThreadsRequest{
						ViewerEntityID: output.entity.ID,
						ThreadIDs:      output.threadIDs,
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					transformedThreads := make([]*models.Thread, len(res.Threads))
					for i, thread := range res.Threads {
						transformedThreads[i], err = transformThreadToResponse(ctx, ram, thread, acc)
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}
					}

					return transformedThreads, nil
				}},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*markThreadsAsReadOutput)
			return ok
		},
	},
)

type threadWatermark struct {
	ThreadID             string `gql:"threadID"`
	LastMessageTimestamp int    `gql:"lastMessageTimestamp"`
}

type markThreadsAsReadInput struct {
	ThreadWatermarks []*threadWatermark `gql:"threadWatermarks"`
	OrganizationID   string             `gql:"organizationID,nonempty"`
	Seen             bool               `gql:"seen"`
	ClientMutationID string             `gql:"clientMutationId"`
}

var threadWatermarkType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ThreadWatermark",
		Fields: graphql.InputObjectConfigFieldMap{
			"threadID":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"lastMessageTimestamp": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		},
	},
)

var markThreadsAsReadInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "MarkThreadsAsReadInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"threadWatermarks": &graphql.InputObjectFieldConfig{Type: graphql.NewList(threadWatermarkType)},
			"seen": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "True indicates that the user has read the messages in the thread up to the watermark for each thread in the list.",
			},
			"organizationID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
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
		ram := raccess.ResourceAccess(p)
		acc := gqlctx.Account(ctx)

		input := p.Args["input"].(map[string]interface{})
		var in markThreadsAsReadInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		ent, err := entityInOrgForAccountID(ctx, ram, in.OrganizationID, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		// nothing to do if no threads to mark as read
		if len(in.ThreadWatermarks) == 0 {
			return &markThreadsAsReadOutput{
				Success:          true,
				ClientMutationID: in.ClientMutationID,
			}, nil
		}

		threadWatermarks := make([]*threading.MarkThreadsAsReadRequest_ThreadWatermark, len(in.ThreadWatermarks))
		threadIDs := make([]string, len(in.ThreadWatermarks))
		for i, watermark := range in.ThreadWatermarks {
			threadWatermarks[i] = &threading.MarkThreadsAsReadRequest_ThreadWatermark{
				ThreadID:             watermark.ThreadID,
				LastMessageTimestamp: uint64(watermark.LastMessageTimestamp),
			}
			threadIDs[i] = watermark.ThreadID
		}
		_, err = ram.MarkThreadsAsRead(ctx, &threading.MarkThreadsAsReadRequest{
			ThreadWatermarks: threadWatermarks,
			EntityID:         ent.ID,
			Seen:             in.Seen,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &markThreadsAsReadOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			threadIDs:        threadIDs,
			entity:           ent,
		}, nil
	}),
}
