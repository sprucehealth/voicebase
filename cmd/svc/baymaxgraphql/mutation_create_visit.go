package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type createVisitOutput struct {
	ClientMutationID string        `json:"clientMutationId"`
	Success          bool          `json:"success"`
	ErrorCode        string        `json:"errorCode,omitempty"`
	ErrorMessage     string        `json:"errorMessage"`
	Visit            *models.Visit `json:"visit,omitempty"`
}

type createVisitInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	VisitLayoutID    string `gql:"visitLayoutID,nonempty"`
	OrganizationID   string `gql:"organizationID,nonempty"`
}

var createVisitInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "CreateVisitInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"visitLayoutID":    &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"organizationID":   &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	createVisitErrorCodeVisitLayoutNotFound = "VISIT_LAYOUT_NOT_FOUND"
)

var createVisitErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreateVisitErrorCode",
	Values: graphql.EnumValueConfigMap{
		createVisitErrorCodeVisitLayoutNotFound: &graphql.EnumValueConfig{
			Value:       createVisitErrorCodeVisitLayoutNotFound,
			Description: "Visit layout not found",
		},
	},
})

var createVisitOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CreateVisitPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createVisitErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"visit":            &graphql.Field{Type: visitType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*createVisitOutput)
		return ok
	},
})

var createVisitMutation = &graphql.Field{
	Type: graphql.NewNonNull(createVisitOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createVisitInputType)},
	},
	Resolve: apiaccess.Patient(
		func(p graphql.ResolveParams) (interface{}, error) {
			ram := raccess.ResourceAccess(p)
			ctx := p.Context
			acc := gqlctx.Account(ctx)
			svc := serviceFromParams(p)

			input := p.Args["input"].(map[string]interface{})
			var in createVisitInput
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

			visitLayoutRes, err := ram.VisitLayout(ctx, &layout.GetVisitLayoutRequest{
				ID: in.VisitLayoutID,
			})
			if err != nil {
				if grpc.Code(err) == codes.NotFound {
					return &createVisitOutput{
						Success:          false,
						ErrorCode:        createVisitErrorCodeVisitLayoutNotFound,
						ClientMutationID: in.ClientMutationID,
					}, nil
				}
				return nil, errors.InternalError(ctx, err)
			}

			createVisitRes, err := ram.CreateVisit(ctx, &care.CreateVisitRequest{
				CreatorID:        ent.ID,
				EntityID:         ent.ID,
				PatientInitiated: true,
				OrganizationID:   in.OrganizationID,
				LayoutVersionID:  visitLayoutRes.VisitLayout.Version.ID,
				Name:             visitLayoutRes.VisitLayout.Name,
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			orgEntity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				Key: &directory.LookupEntitiesRequest_EntityID{
					EntityID: in.OrganizationID,
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			visit, err := transformVisitToResponse(ctx, ram, orgEntity, createVisitRes.Visit, visitLayoutRes.VisitLayout.Version, svc.layoutStore)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			analytics.SegmentTrack(&segment.Track{
				Event:  "patient-initiated-visit-created",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"organization_id": in.OrganizationID,
					"visit_layout_id": visitLayoutRes.VisitLayout.Version.ID,
				},
			})

			return &createVisitOutput{
				Success:          true,
				ClientMutationID: in.ClientMutationID,
				Visit:            visit,
			}, nil
		},
	),
}
