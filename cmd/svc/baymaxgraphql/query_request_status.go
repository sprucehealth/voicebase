package main

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	requestStatusPending  = "PENDING"
	requestStatusComplete = "COMPLETE"
)

// requestStatusEnum represents the possible states of the status field on a request status
var requestStatusEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "RequestStatusStatus",
		Values: graphql.EnumValueConfigMap{
			requestStatusPending: &graphql.EnumValueConfig{
				Value: requestStatusPending,
			},
			requestStatusComplete: &graphql.EnumValueConfig{
				Value: requestStatusComplete,
			},
		},
	},
)

var requestStatusType = graphql.NewObject(graphql.ObjectConfig{
	Name: "RequestStatus",
	Interfaces: []*graphql.Interface{
		nodeInterfaceType,
	},
	Fields: graphql.Fields{
		"id":                 &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"type":               &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"status":             &graphql.Field{Type: graphql.NewNonNull(requestStatusEnum)},
		"description":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"errors":             &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		"tasksRequested":     &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"tasksCompleted":     &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"tasksErrored":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"completedTimestamp": &graphql.Field{Type: graphql.Int},
		"createdTimestamp":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})

var requestStatusQuery = &graphql.Field{
	Type: requestStatusType,
	Args: graphql.FieldConfigArgument{
		"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		return lookupRequestStatus(ctx, ram, p.Args["id"].(string))
	}),
}

func lookupRequestStatus(ctx context.Context, ram raccess.ResourceAccessor, id string) (interface{}, error) {
	// TODO: For now we only support querying the status of threading batch jobs. This may become more generic in the future
	resp, err := ram.BatchJobs(ctx, &threading.BatchJobsRequest{
		LookupKey: &threading.BatchJobsRequest_ID{
			ID: id,
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, errors.Errorf("No request status found for ID %s", id)
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	if len(resp.BatchJobs) != 1 {
		return nil, errors.Errorf("Expected 1 result for batch jobs id query %v, but got %d", id, len(resp.BatchJobs))
	}
	return transformRequestStatusToResponse(ctx, resp.BatchJobs[0]), err
}
