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
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type triageVisitOutput struct {
	ClientMutationID string         `json:"clientMutationId"`
	Success          bool           `json:"success"`
	ErrorCode        string         `json:"errorCode,omitempty"`
	ErrorMessage     string         `json:"errorMessage"`
	Thread           *models.Thread `json:"thread,omitempty"`
}

var triageVisitInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "TriageVisitInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
		"uuid":             newUUIDInputField(),
		"visitID":          &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
})

const (
	triageVisitErrorCodeNotFound          = "VISIT_NOT_FOUND"
	triageVisitErrorCodeCannotTriageVisit = "CANNOT_TRIAGE_VISIT"
)

var triageVisitErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "TriageVisitErrorCode",
	Values: graphql.EnumValueConfigMap{
		triageVisitErrorCodeNotFound: &graphql.EnumValueConfig{
			Value:       triageVisitErrorCodeNotFound,
			Description: "Visit not found",
		},
		triageVisitErrorCodeCannotTriageVisit: &graphql.EnumValueConfig{
			Value:       triageVisitErrorCodeCannotTriageVisit,
			Description: "Cannot triage the visit given its current state",
		},
	},
})

var triageVisitOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "TriageVisitPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: triageVisitErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"thread":           &graphql.Field{Type: threadType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*triageVisitOutput)
		return ok
	},
})

var triageVisitMutation = &graphql.Field{
	Type: graphql.NewNonNull(triageVisitOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(triageVisitInputType)},
	},
	Resolve: apiaccess.Authenticated(apiaccess.Patient(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		svc := serviceFromParams(p)
		if acc == nil {
			return nil, errors.ErrNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		visitID := input["visitID"].(string)
		uuid, _ := input["uuid"].(string)

		visitRes, err := ram.Visit(ctx, &care.GetVisitRequest{
			ID: visitID,
		})
		if err != nil {
			if grpc.Code(err) == codes.NotFound {
				return &triageVisitOutput{
					Success:          false,
					ClientMutationID: mutationID,
					ErrorCode:        triageVisitErrorCodeNotFound,
					ErrorMessage:     fmt.Sprintf("visit %s not found", visitID),
				}, nil
			}
			return nil, errors.InternalError(ctx, err)
		} else if visitRes.Visit.Submitted {
			return &triageVisitOutput{
				Success:          false,
				ClientMutationID: mutationID,
				ErrorCode:        triageVisitErrorCodeCannotTriageVisit,
				ErrorMessage:     fmt.Sprintf("cannot triage visit %s that is already submitted ", visitID),
			}, nil
		}

		_, err = ram.TriageVisit(ctx, &care.TriageVisitRequest{
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
			return &triageVisitOutput{
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

		entity, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: visitRes.Visit.EntityID,
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var titleBML bml.BML
		titleBML = append(titleBML, "Warning! Triaged out of visit before completion: ", &bml.Anchor{
			HREF: deeplink.VisitURL(svc.webDomain, thread.ID, visitID),
			Text: visitRes.Visit.Name,
		})

		titleStr, err := titleBML.Format()
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		postMessageRes, err := ram.PostMessage(ctx, &threading.PostMessageRequest{
			ThreadID:     thread.ID,
			FromEntityID: visitRes.Visit.EntityID,
			UUID:         uuid,
			Message: &threading.MessagePost{
				Summary: fmt.Sprintf("%s:%s", entity.Info.DisplayName, " Warning! Triaged out of visit before completion"),
				Title:   titleStr,
			},
		})

		transformedThread, err := transformThreadToResponse(ctx, ram, postMessageRes.Thread, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		return &triageVisitOutput{
			Success:          true,
			Thread:           transformedThread,
			ClientMutationID: mutationID,
		}, nil
	})),
}
