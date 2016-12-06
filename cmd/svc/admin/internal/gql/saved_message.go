package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

// savedMessageType is a type representing an saved message
var savedMessageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SavedMessage",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"title":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"organizationID":  &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organization":    &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: savedMessageOrganizationResolve},
			"creatorEntityID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"creator":         &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: savedMessageCreatorResolve},
			"ownerEntityID":   &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"owner":           &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: savedMessageOwnerResolve},
			"internal":        &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"created":         &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})

func savedMessageOrganizationResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	sm := p.Source.(*models.SavedMessage)
	return getEntity(ctx, client.Directory(p), sm.OrganizationID)
}

func savedMessageCreatorResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	sm := p.Source.(*models.SavedMessage)
	return getEntity(ctx, client.Directory(p), sm.CreatorEntityID)
}

func savedMessageOwnerResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	sm := p.Source.(*models.SavedMessage)
	return getEntity(ctx, client.Directory(p), sm.OwnerEntityID)
}

func getSavedMessagesForEntity(ctx context.Context, threadingCli threading.ThreadsClient, entityID string) ([]*models.SavedMessage, error) {
	resp, err := threadingCli.SavedMessages(ctx, &threading.SavedMessagesRequest{
		By: &threading.SavedMessagesRequest_EntityIDs{
			EntityIDs: &threading.IDList{
				IDs: []string{entityID},
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformSavedMessagesToModel(resp.SavedMessages), nil
}
