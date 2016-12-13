package main

import (
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
)

var visitListType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "VisitList",
		Description: "VisitList contains a list of visits",
		Fields: graphql.Fields{
			"items": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(visitType))},
		},
	},
)

var visitLayoutListType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "VisitLayoutList",
		Description: "VisitLayoutList contains a list of visit layouts",
		Fields: graphql.Fields{
			"items": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(visitLayoutType))},
		},
	},
)

var visitLayoutsForPatientInitiatedVisitsQuery = &graphql.Field{
	Type: visitLayoutListType,
	Args: graphql.FieldConfigArgument{
		"organizationID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Patient(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		svc := serviceFromParams(p)
		organizationID := p.Args["organizationID"].(string)

		booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
			NodeID: organizationID,
			Keys: []*settings.ConfigKey{
				{
					Key: baymaxgraphqlsettings.ConfigKeyPatientInitiatedVisits,
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

		par := conc.NewParallel()
		visitLayouts := make([][]*layout.VisitLayout, len(res.Categories))
		for i, visitCategory := range res.Categories {
			idx := i
			categoryID := visitCategory.ID
			par.Go(func() error {
				res, err := svc.layout.ListVisitLayouts(ctx, &layout.ListVisitLayoutsRequest{
					VisitCategoryID: categoryID,
				})
				if err != nil {
					return errors.Trace(err)
				}

				visitLayouts[idx] = res.VisitLayouts
				return nil
			})
		}

		if err := par.Wait(); err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		allVisitLayouts := make([]*layout.VisitLayout, 0, len(res.Categories)*len(visitLayouts[0]))
		for _, items := range visitLayouts {
			allVisitLayouts = append(allVisitLayouts, items...)
		}
		sort.Sort(byVisitLayoutName(allVisitLayouts))

		transformedVisitLayouts := make([]*models.VisitLayout, len(allVisitLayouts))
		for i, item := range allVisitLayouts {
			transformedVisitLayouts[i] = transformVisitLayoutToResponse(item)
		}

		return &models.VisitLayoutList{
			Items: transformedVisitLayouts,
		}, nil
	}),
}

var patientInitiatedVisitDraftsQuery = &graphql.Field{
	Type: visitListType,
	Args: graphql.FieldConfigArgument{
		"organizationID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Patient(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)
		svc := serviceFromParams(p)
		organizationID := p.Args["organizationID"].(string)
		acc := gqlctx.Account(p.Context)

		ent, err := entityInOrgForAccountID(ctx, ram, organizationID, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		res, err := ram.Visits(ctx, &care.GetVisitsRequest{
			Draft:            true,
			PatientInitiated: true,
			OrganizationID:   organizationID,
			Query: &care.GetVisitsRequest_CreatorID{
				CreatorID: ent.ID,
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		orgEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_EntityID{
				EntityID: organizationID,
			},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		items := make([]*models.Visit, len(res.Visits))
		for i, visit := range res.Visits {
			layoutVersionRes, err := ram.VisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
				ID: visit.LayoutVersionID,
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			items[i], err = transformVisitToResponse(
				ctx,
				ram,
				orgEntity,
				visit,
				layoutVersionRes.VisitLayoutVersion,
				svc.layoutStore)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		return &models.VisitList{
			Items: items}, nil
	}),
}
