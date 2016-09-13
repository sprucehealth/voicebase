package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var updateFollowingForThreadsInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UpdateFollowingForThreadsInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"orgID":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"threadIDs":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.ID)))},
		"following":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var updateFollowingForThreadsErrorCodeEnum = graphql.String

var updateFollowingForThreadsOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "UpdateFollowingForThreadsPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: updateFollowingForThreadsErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"threads":          &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(threadType))},
		"organization": &graphql.Field{
			Type: organizationType,
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
		_, ok := value.(*updateFollowingForThreadsOutput)
		return ok
	},
})

type updateFollowingForThreadsInput struct {
	ClientMutationID string   `gql:"clientMutationId"`
	OrgID            string   `gql:"orgID"`
	ThreadIDs        []string `gql:"threadIDs"`
	Following        bool     `gql:"following"`
}

type updateFollowingForThreadsOutput struct {
	ClientMutationID string           `json:"clientMutationId,omitempty"`
	Success          bool             `json:"success"`
	ErrorCode        string           `json:"errorCode,omitempty"`
	ErrorMessage     string           `json:"errorMessage,omitempty"`
	Threads          []*models.Thread `json:"threads"`

	orgID    string
	entityID string
}

var updateFollowingForThreadsMutation = &graphql.Field{
	Type: graphql.NewNonNull(updateFollowingForThreadsOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateFollowingForThreadsInputType)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		var in updateFollowingForThreadsInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, in.OrgID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, in.OrgID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		threads := make([]*threading.Thread, len(in.ThreadIDs))
		par := conc.NewParallel()
		for i, id := range in.ThreadIDs {
			ix := i
			par.Go(func() error {
				req := &threading.UpdateThreadRequest{
					ActorEntityID: ent.ID,
					ThreadID:      id,
				}
				if in.Following {
					req.AddFollowerEntityIDs = []string{ent.ID}
				} else {
					req.RemoveFollowerEntityIDs = []string{ent.ID}
				}
				res, err := ram.UpdateThread(ctx, req)
				if err != nil {
					return errors.Trace(err)
				}
				threads[ix] = res.Thread
				return nil
			})
		}
		if err := par.Wait(); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		threadsOut := make([]*models.Thread, len(threads))
		for i, t := range threads {
			th, err := transformThreadToResponse(ctx, ram, t, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			threadsOut[i] = th
		}
		if err := hydrateThreads(ctx, ram, threadsOut); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &updateFollowingForThreadsOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Threads:          threadsOut,
			orgID:            in.OrgID,
			entityID:         ent.ID,
		}, nil
	}),
}
