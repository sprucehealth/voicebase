package main

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/graphql"
)

const (
	layoutContainerTypeIntake = "INTAKE"
	layoutContainerTypeReview = "REVIEW"
)

var layoutContainerType = graphql.NewEnum(graphql.EnumConfig{
	Name:        "LayoutContainerType",
	Description: "Type of the layout container to represent a visit",
	Values: graphql.EnumValueConfigMap{
		layoutContainerTypeIntake: &graphql.EnumValueConfig{
			Value:       layoutContainerTypeIntake,
			Description: "Representation of a visit for the patient",
		},
		layoutContainerTypeReview: &graphql.EnumValueConfig{
			Value:       layoutContainerTypeReview,
			Description: "Representation of a visit for the provider",
		},
	},
})

var visitType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Visit",
		Interfaces: []*graphql.Interface{
			nodeInterfaceType,
		},
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"name": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"entity": &graphql.Field{
				Type: graphql.NewNonNull(entityType),
				Resolve: apiaccess.Authenticated(
					func(p graphql.ResolveParams) (interface{}, error) {
						svc := serviceFromParams(p)
						ctx := p.Context
						ram := raccess.ResourceAccess(p)
						visit := p.Source.(*models.Visit)

						e, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
							Key: &directory.LookupEntitiesRequest_EntityID{
								EntityID: visit.EntityID,
							},
							Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
							RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
						})
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}

						entity, err := transformEntityToResponse(ctx, svc.staticURLPrefix, e, devicectx.SpruceHeaders(ctx), gqlctx.Account(ctx))
						if err != nil {
							return nil, errors.InternalError(ctx, err)
						}

						return entity, nil
					},
				),
			},
			"canReview": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Indicates whether or not a provider can review a visit. Returns true only for a provider when the patient has submitted the visit.",
			},
			"canPatientModify": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Indicates whether or not a patient can modify the visit. Returns true only for a patient before the visit has been submitted.",
			},
			"submitted": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "True when patient has submitted visit, false otherwise.",
			},
			"submittedTimestamp": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "timestamp when the visit was submitted",
			},
			"triaged": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "True if a patient was triaged out before visit was completed, false otherwise.",
			},
			"layoutContainer": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Representation of a visit.",
			},
			"layoutContainerType": &graphql.Field{
				Type:        graphql.NewNonNull(layoutContainerType),
				Description: "Type of layout contained within layout container",
			},
		},
	},
)

func lookupVisit(ctx context.Context, svc *service, ram raccess.ResourceAccessor, id string) (*models.Visit, error) {
	res, err := ram.Visit(ctx, &care.GetVisitRequest{
		ID: id,
	})
	if err != nil {
		return nil, err
	}

	layoutVersionRes, err := ram.VisitLayoutVersion(ctx, &layout.GetVisitLayoutVersionRequest{
		ID: res.Visit.LayoutVersionID,
	})
	if err != nil {
		return nil, err
	}

	orgEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: res.Visit.OrganizationID,
		},
	})
	if err != nil {
		return nil, err
	}

	visit, err := transformVisitToResponse(ctx, ram, orgEntity, res.Visit, layoutVersionRes.VisitLayoutVersion, svc.layoutStore)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return visit, nil
}
