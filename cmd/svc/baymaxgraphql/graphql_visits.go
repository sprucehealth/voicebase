package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

const (
	layoutContainerTypeIntake = "INTAKE"
	layoutContainerTypeReview = "REVIEW"
)

var layoutContainerType = graphql.NewEnum(graphql.EnumConfig{
	Name:        "LayoutContainerType",
	Description: "Type of the layout container to represent a visit",
	Values: graphql.EnumValueConfigMap{
		callEntityTypeConnectParties: &graphql.EnumValueConfig{
			Value:       layoutContainerTypeIntake,
			Description: "Representation of a visit for the patient",
		},
		callEntityTypeReturnPhoneNumber: &graphql.EnumValueConfig{
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
			"canReview": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Indicates whether or not a provider can review a visit. Returns true only for a provider when the patient has submitted the visit.",
			},
			"canModify": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Indicates whether or not a patient can modify the visit. Returns true only for a patient before the visit has been submitted.",
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

	visit, err := transformVisitToResponse(ctx, res.Visit, layoutVersionRes.VisitLayoutVersion, svc.layoutStore)
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	return visit, nil
}
