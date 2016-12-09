package gql

import (
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/graphql"
)

// triggeredMessageItemType is a type representing an triggered message
var triggeredMessageItemType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "TriggeredMessageItem",
		Fields: graphql.Fields{
			"id":                 &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"triggeredMessageID": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"actorEntityID":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"actor":              &graphql.Field{Type: graphql.NewNonNull(entityType), Resolve: triggeredMessageItemActorResolve},
			"internal":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"ordinal":            &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"created":            &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"message":            &graphql.Field{Type: graphql.NewNonNull(messageType)},
		},
	})

func triggeredMessageItemActorResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	tm := p.Source.(*models.TriggeredMessageItem)
	return getEntity(ctx, client.Directory(p), tm.ActorEntityID)
}
