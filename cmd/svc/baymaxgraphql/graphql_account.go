package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var meType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Me",
		Fields: graphql.Fields{
			"account":             &graphql.Field{Type: graphql.NewNonNull(accountInterfaceType)},
			"clientEncryptionKey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var accountInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "Account",
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"organizations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(organizationType))},
		},
	},
)

func init() {
	// This is done here rather than at declaration time to avoid an unresolvable compile time decleration loop
	accountInterfaceType.ResolveType = func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
		switch value.(type) {
		case *models.ProviderAccount:
			return providerAccountType
		}
		return nil
	}
}

func lookupAccount(ctx context.Context, ram raccess.ResourceAccessor, accountID string) (interface{}, error) {
	account, err := ram.Account(ctx, accountID)
	if err != nil {
		return nil, err
	}
	// Since we only use the ID we don't really need to do the lookup, but
	// it allows us to check if the account exists.
	return &models.ProviderAccount{
		ID: account.ID,
	}, nil
}
