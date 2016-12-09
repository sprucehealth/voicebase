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
)

var updateThreadInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateThreadInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId":        newClientMutationIDInputField(),
		"uuid":                    newUUIDInputField(),
		"threadID":                &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"addMemberEntityIDs":      &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"removeMemberEntityIDs":   &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"addFollowerEntityIDs":    &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"removeFollowerEntityIDs": &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"addTags":                 &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		"removeTags":              &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		"title":                   &graphql.InputObjectFieldConfig{Type: graphql.String},
	},
})

const (
	updateThreadErrorCodeInvalidTag = "INVALID_TAG"
)

var updateThreadErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "UpdateThreadErrorCode",
	Description: "Result of updateThread mutation",
	Values: graphql.EnumValueConfigMap{
		updateThreadErrorCodeInvalidTag: &graphql.EnumValueConfig{
			Value:       updateThreadErrorCodeInvalidTag,
			Description: "A provided tag is invalid",
		},
	},
})

var updateThreadOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateThreadPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateThreadErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread":           &graphql.Field{Type: graphql.NewNonNull(threadType)},
		"organization": &graphql.Field{
			Type: graphql.NewList(graphql.NewNonNull(organizationType)),
			Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				svc := serviceFromParams(p)
				out := p.Source.(*updateFollowingForThreadsOutput)
				return lookupEntity(ctx, svc, ram, out.orgID)
			}),
		},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*updateThreadOutput)
		return ok
	},
})

type updateThreadInput struct {
	ClientMutationID        string   `gql:"clientMutationId"`
	ThreadID                string   `gql:"threadID"`
	Title                   string   `gql:"title"`
	AddMemberEntityIDs      []string `gql:"addMemberEntityIDs"`
	RemoveMemberEntityIDs   []string `gql:"removeMemberEntityIDs"`
	AddFollowerEntityIDs    []string `gql:"addFollowerEntityIDs"`
	RemoveFollowerEntityIDs []string `gql:"removeFollowerEntityIDs"`
	AddTags                 []string `gql:"addTags"`
	RemoveTags              []string `gql:"removeTags"`
}

type updateThreadOutput struct {
	ClientMutationID string         `json:"clientMutationId,omitempty"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	Thread           *models.Thread `json:"thread"`

	orgID    string
	entityID string
}

var updateThreadMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateThreadOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateThreadInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		var in updateThreadInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		thread, err := ram.Thread(ctx, in.ThreadID, "")
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if thread == nil {
			return nil, errors.ErrNotFound(ctx, in.ThreadID)
		}
		if thread.Type != threading.THREAD_TYPE_TEAM {
			if len(in.AddMemberEntityIDs) != 0 {
				return nil, errors.New("Cannot modify members on non-team threads")
			}
			if len(in.RemoveMemberEntityIDs) != 0 {
				return nil, errors.New("Cannot modify members on non-team threads")
			}
			if in.Title != "" {
				return nil, errors.New("Cannot modify title on non-team threads")
			}
		}
		for _, t := range append(in.AddTags, in.RemoveTags...) {
			if !threading.ValidateTag(t, false) {
				return &updateThreadOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        updateThreadErrorCodeInvalidTag,
					ErrorMessage:     fmt.Sprintf("%q is not a valid tag. Tags must only contain characters, numbers, underscores, and dashes.", t),
				}, nil
			}
		}

		// TODO: currently assuming that the person updating the thread is in the same org as the thread.
		//       This is safe for now, but possibly may not be true in the future.
		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, thread.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, thread.OrganizationID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		res, err := ram.UpdateThread(ctx, &threading.UpdateThreadRequest{
			ActorEntityID:           ent.ID,
			ThreadID:                thread.ID,
			UserTitle:               in.Title,
			AddMemberEntityIDs:      in.AddMemberEntityIDs,
			RemoveMemberEntityIDs:   in.RemoveMemberEntityIDs,
			AddFollowerEntityIDs:    in.AddFollowerEntityIDs,
			RemoveFollowerEntityIDs: in.RemoveFollowerEntityIDs,
			AddTags:                 in.AddTags,
			RemoveTags:              in.RemoveTags,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		// The thread will be nil in the response if the viewer removed them self as a member of a team thread
		if res.Thread == nil {
			return &updateThreadOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          true,
				orgID:            thread.OrganizationID,
				entityID:         ent.ID,
			}, nil
		}
		thread = res.Thread

		th, err := transformThreadToResponse(ctx, ram, thread, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		if err := hydrateThreads(ctx, ram, []*models.Thread{th}); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &updateThreadOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Thread:           th,
			orgID:            thread.OrganizationID,
			entityID:         ent.ID,
		}, nil
	}),
}
