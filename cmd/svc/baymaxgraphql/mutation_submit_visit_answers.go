package main

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/graphql"
)

type submitVisitAnswersOutput struct {
	ClientMutationID string   `json:"clientMutationId,omitempty"`
	Success          bool     `json:"success"`
	ErrorCode        string   `json:"errorCode,omitempty"`
	ErrorMessage     string   `json:"errorMessage"`
	ErrorQuestionIDs []string `json:"errorQuestionIDs,omitempty"`
}

var submitVisitAnswersInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "SubmitVisitAnswersInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"visitID":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"answersJSON":      &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
	},
})

const (
	submitVisitErrorCodeCannotModifyVisit = "CANNOT_MODIFY_VISIT"
	submitVisitErrorCodeInvalidAnswer     = "INVALID_ANSWER"
)

var submitVisitAnswersErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "SubmitVisitAnswersErrorCode",
	Values: graphql.EnumValueConfigMap{
		submitVisitErrorCodeInvalidAnswer: &graphql.EnumValueConfig{
			Value:       submitVisitErrorCodeInvalidAnswer,
			Description: "The provided answer is invalid",
		},
		submitVisitErrorCodeCannotModifyVisit: &graphql.EnumValueConfig{
			Value:       submitVisitErrorCodeCannotModifyVisit,
			Description: "Cannot modify a visit once it has been submitted",
		},
	},
})

var submitVisitAnswersOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SubmitVisitAnswersPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: submitVisitAnswersErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"errorQuestionIDs": &graphql.Field{Type: graphql.NewList(graphql.String)},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*submitVisitAnswersOutput)
		return ok
	},
})

var submitVisitAnswersMutation = &graphql.Field{
	Type: graphql.NewNonNull(submitVisitAnswersOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(submitVisitAnswersInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Patient(
			func(p graphql.ResolveParams) (interface{}, error) {
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)

				if acc == nil {
					return nil, errors.ErrNotAuthenticated(ctx)
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				visitID, _ := input["visitID"].(string)
				answersJSON, _ := input["answersJSON"].(string)

				visitRes, err := ram.Visit(ctx, &care.GetVisitRequest{
					ID: visitID,
				})
				if err != nil {
					return nil, err
				}

				if visitRes.Visit.Submitted {
					return &submitVisitAnswersOutput{
						Success:          false,
						ErrorCode:        submitVisitErrorCodeCannotModifyVisit,
						ErrorMessage:     fmt.Sprintf("Cannot modify visit %s as it is already submitted", visitRes.Visit.ID),
						ClientMutationID: mutationID,
					}, nil
				}

				_, err = ram.CreateVisitAnswers(ctx, &care.CreateVisitAnswersRequest{
					ActorEntityID: visitRes.Visit.EntityID,
					AnswersJSON:   answersJSON,
					VisitID:       visitRes.Visit.ID,
				})
				if err != nil {
					if grpc.Code(err) == care.ErrorInvalidAnswer {
						return &submitVisitAnswersOutput{
							Success:      false,
							ErrorCode:    submitVisitErrorCodeInvalidAnswer,
							ErrorMessage: fmt.Sprintf("Invalid answer for visit %s", visitRes.Visit.ID),
						}, nil
					}
					return nil, errors.InternalError(ctx, err)
				}

				return &submitVisitAnswersOutput{
					Success:          true,
					ClientMutationID: mutationID,
				}, nil
			},
		),
	),
}
