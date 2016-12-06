package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// triggeredMessageType is a type representing an triggered message
var triggeredMessageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "TriggeredMessage",
		Fields: graphql.Fields{
			"id": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizationEntityID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organization":         &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: triggeredMessageOrganizationResolve},
			"actorEntityID":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"actor":                &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: triggeredMessageActorResolve},
			"key":                  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"subkey":               &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"enabled":              &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"created":              &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})

func triggeredMessageOrganizationResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	tm := p.Source.(*models.TriggeredMessage)
	return getEntity(ctx, client.Directory(p), tm.OrganizationEntityID)
}

func triggeredMessageActorResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	tm := p.Source.(*models.TriggeredMessage)
	return getEntity(ctx, client.Directory(p), tm.ActorEntityID)
}

func getTriggeredMessage(ctx context.Context, threadingCli threading.ThreadsClient, key threading.TriggeredMessageKey_Key, subkey string) ([]*models.TriggeredMessage, error) {
	resp, err := threadingCli.TriggeredMessages(ctx, &threading.TriggeredMessagesRequest{
		LookupKey: &threading.TriggeredMessagesRequest_Key{
			Key: &threading.TriggeredMessageKey{
				Key:    key,
				Subkey: subkey,
			},
		},
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformTriggeredMessagesToModel(resp.TriggeredMessages), nil
}
