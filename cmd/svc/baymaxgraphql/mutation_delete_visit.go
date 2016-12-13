package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type deleteVisitOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage"`
}

type deleteVisitInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	VisitID          string `gql:"visitID"`
}

var deleteVisitInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "DeleteVisitInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"visitID":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	deleteVisitErrorCodeVisitAlreadySubmitted = "VISIT_ALREADY_SUBMITTED"
	deleteVisitErrorCodeNotFound              = "VISIT_NOT_FOUND"
)

var deleteVisitErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "DeleteVisitErrorCode",
	Values: graphql.EnumValueConfigMap{
		deleteVisitErrorCodeVisitAlreadySubmitted: &graphql.EnumValueConfig{
			Value:       deleteVisitErrorCodeVisitAlreadySubmitted,
			Description: "Visit already submitted",
		},
		deleteVisitErrorCodeNotFound: &graphql.EnumValueConfig{
			Value:       deleteVisitErrorCodeNotFound,
			Description: "Visit not found",
		},
	},
})

var deleteVisitOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "DeleteVisitPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: createVisitErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*deleteVisitOutput)
		return ok
	},
})

var deleteVisitMutation = &graphql.Field{
	Type: graphql.NewNonNull(deleteVisitOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deleteVisitInputType)},
	},
	Resolve: apiaccess.Patient(
		func(p graphql.ResolveParams) (interface{}, error) {
			ram := raccess.ResourceAccess(p)
			ctx := p.Context
			acc := gqlctx.Account(ctx)

			input := p.Args["input"].(map[string]interface{})
			var in deleteVisitInput
			if err := gqldecode.Decode(input, &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}

			visitRes, err := ram.Visit(ctx, &care.GetVisitRequest{
				ID: in.VisitID,
			})
			if err != nil {
				if grpc.Code(err) == codes.NotFound {
					return &deleteVisitOutput{
						Success:          false,
						ErrorCode:        deleteVisitErrorCodeNotFound,
						ClientMutationID: in.ClientMutationID,
					}, nil
				}
				return nil, errors.InternalError(ctx, err)
			} else if visitRes.Visit.Submitted {
				return &deleteVisitOutput{
					Success:          false,
					ErrorCode:        deleteVisitErrorCodeVisitAlreadySubmitted,
					ClientMutationID: in.ClientMutationID,
				}, nil
			}

			ent, err := entityInOrgForAccountID(ctx, ram, visitRes.Visit.OrganizationID, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			if _, err := ram.DeleteVisit(ctx, &care.DeleteVisitRequest{
				ID:            in.VisitID,
				ActorEntityID: ent.ID,
			}); err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			analytics.SegmentTrack(&segment.Track{
				Event:  "patient-initiated-visit-deleted",
				UserId: acc.ID,
				Properties: map[string]interface{}{
					"organization_id": visitRes.Visit.OrganizationID,
					"visit_id":        visitRes.Visit.ID,
				},
			})

			return &deleteVisitOutput{
				Success:          true,
				ClientMutationID: in.ClientMutationID,
			}, nil
		},
	),
}
