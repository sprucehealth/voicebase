package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type submitVisitOutput struct {
	ClientMutationID string         `json:"clientMutationId"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage"`
	ErrorQuestionIDs []string       `json:"errorQuestionIDs,omitempty"`
	Thread           *models.Thread `json:"thread,omitempty"`
}

var submitVisitInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "SubmitVisitInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"visitID":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	submitVisitErrorCodeNotFound = "VISIT_NOT_FOUND"
)

var submitVisitErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "SubmitVisitErrorCode",
	Values: graphql.EnumValueConfigMap{
		submitVisitErrorCodeNotFound: &graphql.EnumValueConfig{
			Value:       submitVisitErrorCodeNotFound,
			Description: "Visit not found",
		},
	},
})

var submitVisitOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SubmitVisitPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: submitVisitErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"errorQuestionIDs": &graphql.Field{Type: graphql.NewList(graphql.String)},
		"thread":           &graphql.Field{Type: threadType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*submitVisitOutput)
		return ok
	},
})

var submitVisitMutation = &graphql.Field{
	Type: graphql.NewNonNull(submitVisitOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(submitVisitInputType)},
	},
	Resolve: apiaccess.Authenticated(
		apiaccess.Patient(
			func(p graphql.ResolveParams) (interface{}, error) {
				ram := raccess.ResourceAccess(p)
				ctx := p.Context
				acc := gqlctx.Account(ctx)
				svc := serviceFromParams(p)
				if acc == nil {
					return nil, errors.ErrNotAuthenticated(ctx)
				}

				input := p.Args["input"].(map[string]interface{})
				mutationID, _ := input["clientMutationId"].(string)
				visitID, _ := input["visitID"].(string)
				uuid, _ := input["uuid"].(string)

				// get the visit so that we have the entity and can look up the
				// thread in which to post the message
				visitRes, err := ram.Visit(ctx, &care.GetVisitRequest{
					ID: visitID,
				})
				if err != nil {
					if grpc.Code(err) == codes.NotFound {
						return &submitVisitOutput{
							Success:          false,
							ClientMutationID: mutationID,
							ErrorCode:        submitVisitErrorCodeNotFound,
							ErrorMessage:     fmt.Sprintf("visit %s not found", visitID),
						}, nil
					}
					return nil, errors.InternalError(ctx, err)
				}

				_, err = ram.SubmitVisit(ctx, &care.SubmitVisitRequest{
					VisitID: visitID,
				})
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				// lookup threads that this patient owns
				// note it should be just once
				threads, err := ram.ThreadsForMember(ctx, visitRes.Visit.EntityID, true)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				} else if len(threads) == 0 {
					return &submitVisitOutput{
						ClientMutationID: mutationID,
						Success:          true,
					}, nil
				}

				// identify the thread to post message in
				var thread *threading.Thread
				for _, th := range threads {
					if th.OrganizationID == visitRes.Visit.OrganizationID {
						thread = th
						break
					}
				}
				if thread == nil {
					return nil, errors.InternalError(ctx, fmt.Errorf("thread in organization %s not found", visitRes.Visit.OrganizationID))
				}

				entity, err := ram.Entity(ctx, visitRes.Visit.EntityID, nil, 0)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				var titleBML bml.BML
				titleBML = append(titleBML, "Completed a visit: ", &bml.Ref{
					Type: bml.AttachmentRef,
					URL:  deeplink.VisitURL(svc.webDomain, thread.ID, visitID),
					ID:   visitID,
					Text: visitRes.Visit.Name,
				})

				titleStr, err := titleBML.Format()
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}

				postMessageRes, err := ram.PostMessage(ctx, &threading.PostMessageRequest{
					ThreadID:     thread.ID,
					FromEntityID: visitRes.Visit.EntityID,
					Summary:      fmt.Sprintf("%s:%s", entity.Info.DisplayName, "Completed a visit"),
					UUID:         uuid,
					Title:        titleStr,
				})

				transformedThread, err := transformThreadToResponse(ctx, ram, postMessageRes.Thread, acc)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				return &submitVisitOutput{
					Success:          true,
					Thread:           transformedThread,
					ClientMutationID: mutationID,
				}, nil
			},
		),
	),
}
