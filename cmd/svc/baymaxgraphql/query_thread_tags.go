package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"

	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var threadTagListType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "ThreadTagList",
		Description: "ThreadTagList contains a list of available thread tags",
		Fields: graphql.Fields{
			"items": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		},
	},
)

var threadTagsQuery = &graphql.Field{
	Type: threadTagListType,
	Args: graphql.FieldConfigArgument{
		"organizationID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		organizationID := p.Args["organizationID"].(string)
		acc := gqlctx.Account(p.Context)
		ram := raccess.ResourceAccess(p)

		if acc.Type == auth.AccountType_PATIENT {
			return nil, nil
		}

		res, err := ram.Tags(ctx, &threading.TagsRequest{
			OrganizationID: organizationID,
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		var tagList models.ThreadTagList
		tagList.Items = make([]string, 0, len(res.Tags))
		for _, tag := range res.Tags {
			if !tag.Hidden {
				tagList.Items = append(tagList.Items, tag.Name)
			}
		}

		return &tagList, nil
	}),
}
