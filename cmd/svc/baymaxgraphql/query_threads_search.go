package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

const maxThreadSearchResults = 500

type threadsSearchInput struct {
	OrganizationID string `gql:"organizationID"`
	Query          string `gql:"query"`
}

var threadsSearchQuery = &graphql.Field{
	Type: graphql.NewNonNull(threadConnectionType.ConnectionType),
	Args: graphql.FieldConfigArgument{
		"organizationID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
		"query":          &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)
		acc := gqlctx.Account(ctx)

		var in threadsSearchInput
		if err := gqldecode.Decode(p.Args, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		query, err := threading.ParseQuery(in.Query)
		if err != nil {
			return nil, gqlerrors.FormatError(errors.New("Your query is invalid"))
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
			Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		}, in.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, in.OrganizationID)
		} else if err != nil {
			return nil, errors.Trace(err)
		}

		var org *directory.Entity
		for _, em := range ent.Memberships {
			if em.Type == directory.EntityType_ORGANIZATION {
				if org != nil {
					return nil, errors.InternalError(ctx, fmt.Errorf("Expected only one org for entity %s", ent.ID))
				}
				org = em
			}
		}
		if org == nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("No organizations for entity %s", ent.ID))
		}

		res, err := ram.QueryThreads(ctx, &threading.QueryThreadsRequest{
			ViewerEntityID: ent.ID,
			Iterator: &threading.Iterator{
				Direction: threading.ITERATOR_DIRECTION_FROM_START,
				Count:     maxThreadSearchResults,
			},
			Type: threading.QUERY_THREADS_TYPE_ADHOC,
			QueryType: &threading.QueryThreadsRequest_Query{
				Query: query,
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return transformQueryThreadsResponseToConnection(ctx, ram, acc, res)
	}),
}
