package gql

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
)

// searchResultsType is a type representing the results of a search request
var searchResultsType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "SearchResults",
		Fields: graphql.Fields{
			"accounts": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(accountType))},
			"entities": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(entityType))},
		},
	})

type searchInput struct {
	Text string `gql:"text,nonempty"`
}

var searchInputType = graphql.FieldConfigArgument{
	"text": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
}

// searchField represents a graphql field for searching the backend
var searchField = &graphql.Field{
	Type:    searchResultsType,
	Args:    searchInputType,
	Resolve: searchResolve,
}

func searchResolve(p graphql.ResolveParams) (interface{}, error) {
	ctx := p.Context
	var in searchInput
	if err := gqldecode.Decode(p.Args, &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}
	golog.ContextLogger(ctx).Debugf("Performing Search with args %+v", in)

	return search(p.Context, client.Auth(p), client.Directory(p), &in)
}

func search(ctx context.Context, authClient auth.AuthClient, dirClient directory.DirectoryClient, in *searchInput) (*models.SearchResults, error) {
	in.Text = strings.TrimSpace(in.Text)
	// This should be enforced by the API but be defensive since it's cheap
	if in.Text == "" {
		return &models.SearchResults{}, nil
	}

	// Short Circuit obvious ids we support
	switch {
	case strings.HasPrefix(in.Text, auth.AccountIDPrefix):
		acc, err := getAccountByID(ctx, authClient, in.Text)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &models.SearchResults{
			Accounts: []*models.Account{acc},
		}, nil
	case strings.HasPrefix(in.Text, directory.EntityIDPrefix):
		ent, err := getEntity(ctx, dirClient, in.Text)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &models.SearchResults{
			Entities: []*models.Entity{ent},
		}, nil
	}

	// Perform all other loookup types in parallel
	// TODO: Dedupe results from multiple lookup dimensions for the same type
	sr := &models.SearchResults{}
	parallel := conc.NewParallel()
	parallel.Go(func() error {
		if err := searchEntities(ctx, dirClient, sr, in); err != nil {
			return errors.Trace(err)
		}
		return nil
	})
	parallel.Go(func() error {
		if err := searchAccounts(ctx, authClient, sr, in); err != nil {
			return errors.Trace(err)
		}
		return nil
	})
	if err := parallel.Wait(); err != nil {
		return nil, errors.Trace(err)
	}
	return sr, nil
}

func searchEntities(ctx context.Context, dirClient directory.DirectoryClient, sr *models.SearchResults, in *searchInput) error {
	// TODO: Dedupe
	parallel := conc.NewParallel()
	parallel.Go(func() error {
		displayNameSearchResults, err := dirClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_DisplayName{
				DisplayName: in.Text,
			},
			// TODO: Eventually this should be paramed
			RootTypes: []directory.EntityType{
				directory.EntityType_ORGANIZATION,
				directory.EntityType_INTERNAL,
			},
		})
		if grpc.Code(err) == codes.NotFound {
			return nil
		} else if err != nil {
			return errors.Trace(err)
		}
		if len(displayNameSearchResults.Entities) != 0 {
			sr.Entities = append(sr.Entities, models.TransformEntitiesToModels(displayNameSearchResults.Entities)...)
		}
		return nil
	})
	return errors.Trace(parallel.Wait())
}

func searchAccounts(ctx context.Context, authClient auth.AuthClient, sr *models.SearchResults, in *searchInput) error {
	// TODO: Dedupe
	parallel := conc.NewParallel()
	parallel.Go(func() error {
		account, err := getAccountByEmail(ctx, authClient, in.Text)
		if grpc.Code(errors.Cause(err)) == codes.NotFound {
			return nil
		} else if err != nil {
			return errors.Trace(err)
		}
		sr.Accounts = append(sr.Accounts, account)
		return nil
	})
	return errors.Trace(parallel.Wait())
}
