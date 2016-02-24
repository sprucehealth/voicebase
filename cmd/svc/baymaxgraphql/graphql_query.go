package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type: graphql.NewNonNull(meType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					acc := gqlctx.Account(p.Context)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(p.Context)
					}
					cek := gqlctx.ClientEncryptionKey(p.Context)
					return &models.Me{Account: acc, ClientEncryptionKey: cek}, nil
				},
			},
			"node": &graphql.Field{
				Type: graphql.NewNonNull(nodeInterfaceType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					svc := serviceFromParams(p)
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					id := p.Args["id"].(string)
					prefix := nodePrefix(id)
					switch prefix {
					case "entity":
						return lookupEntity(ctx, ram, id)
					case "account":
						return lookupAccount(ctx, ram, id)
					case "sq":
						return lookupSavedQuery(ctx, ram, id)
					case "t":
						return lookupThreadWithReadStatus(ctx, ram, acc, id)
					case "ti":
						return lookupThreadItem(ctx, ram, svc.mediaSigner, id)
					}
					return nil, errors.New("unknown node type")
				},
			},
			"organization": &graphql.Field{
				Type: graphql.NewNonNull(organizationType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)

					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					return lookupEntity(ctx, ram, p.Args["id"].(string))
				},
			},
			"savedThreadQuery": &graphql.Field{
				Type: graphql.NewNonNull(savedThreadQueryType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					return lookupSavedQuery(ctx, ram, p.Args["id"].(string))
				},
			},
			"thread": &graphql.Field{
				Type: graphql.NewNonNull(threadType),
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}
					it, err := lookupThreadWithReadStatus(ctx, ram, acc, p.Args["id"].(string))
					return it, err
				},
			},
			"subdomain": &graphql.Field{
				Type: graphql.NewNonNull(subdomainType),
				Args: graphql.FieldConfigArgument{
					"value": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ram := raccess.ResourceAccess(p)
					ctx := p.Context
					acc := gqlctx.Account(ctx)
					domain := p.Args["value"].(string)
					if acc == nil {
						return nil, errors.ErrNotAuthenticated(ctx)
					}

					var available bool
					_, err := ram.EntityDomain(ctx, "", domain)
					if grpc.Code(err) == codes.NotFound {
						available = true
					} else if err != nil {
						return nil, err
					}

					return &models.Subdomain{
						Available: available,
					}, nil
				},
			},
			"setting":            settingsQuery,
			"forceUpgradeStatus": forceUpgradeQuery,
		},
	},
)

// TODO: This double read is inefficent/incorrect in the sense that we need the org ID to get the correct entity. We will use this for now until we can encode the organization ID into the thread ID
func lookupThreadWithReadStatus(ctx context.Context, ram raccess.ResourceAccessor, acc *models.Account, id string) (interface{}, error) {
	th, err := lookupThread(ctx, ram, id, "")
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}
	ent, err := ram.EntityForAccountID(ctx, th.OrganizationID, acc.ID)
	if errors.Type(err) == errors.ErrTypeNotFound {
		return nil, errors.UserError(ctx, errors.ErrTypeNotAuthorized, "You are not a member of the organzation")
	} else if err != nil {
		return nil, err
	}
	return lookupThread(ctx, ram, id, ent.ID)
}
