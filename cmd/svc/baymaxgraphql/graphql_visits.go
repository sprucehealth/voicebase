package main

import (
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var visitCategoryConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "VisitCategory",
	NodeType: visitCategoryType,
})

var visitCategoriesField = &graphql.Field{
	Type: visitCategoryConnectionType.ConnectionType,
	Args: NewConnectionArguments(nil),
	Resolve: apiaccess.Authenticated(
		apiaccess.Provider(
			func(p graphql.ResolveParams) (interface{}, error) {
				ctx := p.Context
				svc := serviceFromParams(p)
				org := p.Source.(*models.Organization)

				// TODO: Organization specific pathways
				// TODO: VisitCategory pagination

				// return nothing if disabled for org
				booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
					NodeID: org.ID,
					Keys: []*settings.ConfigKey{
						{
							Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
						},
					},
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				if !booleanValue.Value {
					return nil, nil
				}

				res, err := svc.layout.ListVisitCategories(ctx, &layout.ListVisitCategoriesRequest{})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				sort.Sort(byVisitCategoryName(res.Categories))

				cn := &Connection{
					Edges: make([]*Edge, len(res.Categories)),
				}
				for i, visitCategory := range res.Categories {
					cn.Edges[i] = &Edge{
						Node:   transformVisitCategoryToResponse(visitCategory),
						Cursor: ConnectionCursor(visitCategory.ID),
					}
				}

				return cn, nil
			},
		)),
}

var visitLayoutConnectionType = ConnectionDefinitions(ConnectionConfig{
	Name:     "VisitLayout",
	NodeType: visitLayoutType,
})

var visitCategoryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VisitCategory",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"visitLayouts": &graphql.Field{
				Type: graphql.NewNonNull(visitLayoutConnectionType.ConnectionType),
				Args: NewConnectionArguments(nil),
				Resolve: apiaccess.Authenticated(
					apiaccess.Provider(
						func(p graphql.ResolveParams) (interface{}, error) {
							// TODO: VisitLayout pagination{
							ctx := p.Context
							svc := serviceFromParams(p)
							category := p.Source.(*models.VisitCategory)

							res, err := svc.layout.ListVisitLayouts(ctx, &layout.ListVisitLayoutsRequest{
								VisitCategoryID: category.ID,
							})
							if err != nil {
								return nil, errors.InternalError(ctx, err)
							}
							sort.Sort(byVisitLayoutName(res.VisitLayouts))

							cn := &Connection{
								Edges: make([]*Edge, len(res.VisitLayouts)),
							}
							for i, visitLayout := range res.VisitLayouts {
								cn.Edges[i] = &Edge{
									Node:   transformVisitLayoutToResponse(visitLayout),
									Cursor: ConnectionCursor(visitLayout.ID),
								}
							}

							return cn, nil
						},
					)),
			},
		},
	},
)

var visitLayoutType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VisitLayout",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"version": &graphql.Field{
				Type: graphql.NewNonNull(visitLayoutVersionType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context
					svc := serviceFromParams(p)
					visitLayout := p.Source.(*models.VisitLayout)

					res, err := svc.layout.GetVisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
						VisitLayoutID: visitLayout.ID,
					})
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					tVersion, err := transformVisitLayoutVersionToResponse(res.VisitLayoutVersion, svc.layoutStore)
					if err != nil {
						return nil, errors.InternalError(ctx, err)
					}

					return tVersion, nil
				}},
		},
	},
)

var visitLayoutVersionType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VisitLayoutVersion",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"samlLayout":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"layoutPreview": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

func lookupVisitLayout(ctx context.Context, svc *service, id string) (*models.VisitLayout, error) {
	res, err := svc.layout.GetVisitLayout(ctx, &layout.GetVisitLayoutRequest{
		ID: id,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return transformVisitLayoutToResponse(res.VisitLayout), nil
}

func lookupVisitCategory(ctx context.Context, svc *service, id string) (*models.VisitCategory, error) {
	res, err := svc.layout.GetVisitCategory(ctx, &layout.GetVisitCategoryRequest{
		ID: id,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return transformVisitCategoryToResponse(res.VisitCategory), nil
}

func lookupVisitLayoutVersion(ctx context.Context, svc *service, id string) (*models.VisitLayoutVersion, error) {
	res, err := svc.layout.GetVisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
		ID: id,
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	visitLayoutVersion, err := transformVisitLayoutVersionToResponse(res.VisitLayoutVersion, svc.layoutStore)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return visitLayoutVersion, nil
}
