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

var patientAccountType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PatientAccount",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
			accountInterfaceType,
		},
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type": &graphql.Field{
				Type: graphql.NewNonNull(accountTypeEnum),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return string(models.AccountTypePatient), nil
				},
			},
			"entity": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return patientEntity(p, p.Source.(*models.PatientAccount))
				},
			},
			"accountEntity": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return patientEntity(p, p.Source.(*models.PatientAccount))
				},
			},
			"organizations": &graphql.Field{
				Type: graphql.NewList(graphql.NewNonNull(organizationType)),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return accountOrganizations(p, p.Source.(*models.PatientAccount))
				},
			},
			"threads": &graphql.Field{
				Type: threadConnectionType.ConnectionType,
				Args: NewConnectionArguments(nil),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return patientThreads(p, p.Source.(*models.PatientAccount))
				},
			},
		},
	},
)

func patientThreads(p graphql.ResolveParams, a *models.PatientAccount) (*Connection, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	if gqlctx.Account(ctx) == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: a.GetID(),
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	req := &threading.QueryThreadsRequest{
		Type:           threading.QUERY_THREADS_TYPE_ALL_FOR_VIEWER,
		Iterator:       &threading.Iterator{},
		ViewerEntityID: ent.ID,
	}
	if s, ok := p.Args["after"].(string); ok {
		req.Iterator.StartCursor = s
	}
	if s, ok := p.Args["before"].(string); ok {
		req.Iterator.EndCursor = s
	}
	if i, ok := p.Args["last"].(int); ok {
		req.Iterator.Count = uint32(i)
		req.Iterator.Direction = threading.ITERATOR_DIRECTION_FROM_END
	} else if i, ok := p.Args["first"].(int); ok {
		req.Iterator.Count = uint32(i)
		req.Iterator.Direction = threading.ITERATOR_DIRECTION_FROM_START
	}
	res, err := ram.QueryThreads(ctx, req)
	if err != nil {
		return nil, err
	}

	cn := &Connection{
		Edges: make([]*Edge, len(res.Edges)),
	}
	if req.Iterator.Direction == threading.ITERATOR_DIRECTION_FROM_START {
		cn.PageInfo.HasNextPage = res.HasMore
	} else {
		cn.PageInfo.HasPreviousPage = res.HasMore
	}
	threads := make([]*models.Thread, len(res.Edges))
	for i, e := range res.Edges {
		t, err := transformThreadToResponse(ctx, ram, e.Thread, gqlctx.Account(ctx))
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("Failed to transform thread: %s", err))
		}
		threads[i] = t
		cn.Edges[i] = &Edge{
			Node:   t,
			Cursor: ConnectionCursor(e.Cursor),
		}
	}

	if err := hydrateThreads(ctx, ram, threads); err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return cn, nil
}

func patientEntity(p graphql.ResolveParams, a *models.PatientAccount) (*models.Entity, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	svc := serviceFromParams(p)
	if gqlctx.Account(ctx) == nil {
		return nil, errors.ErrNotAuthenticated(ctx)
	}
	entities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: a.GetID(),
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	} else if len(entities) != 1 {
		return nil, fmt.Errorf("expected 1 entity for %s but got %d back", a.GetID(), len(entities))
	}

	return transformEntityToResponse(
		ctx,
		svc.staticURLPrefix,
		entities[0],
		devicectx.SpruceHeaders(ctx),
		gqlctx.Account(ctx))
}
