package main

import (
	"fmt"
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

var savedMessageType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SavedMessage",
	Fields: graphql.Fields{
		"id":         &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"title":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"shared":     &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"threadItem": &graphql.Field{Type: graphql.NewNonNull(threadItemType)},
	},
})

var savedMessageSectionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SavedMessageSection",
	Fields: graphql.Fields{
		"title":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"messages": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(savedMessageType))},
	},
})

type savedMessagesQueryInput struct {
	OrganizationID string `gql:"organizationID"`
}

var savedMessagesQuery = &graphql.Field{
	Type: graphql.NewList(graphql.NewNonNull(savedMessageSectionType)),
	Args: graphql.FieldConfigArgument{
		"organizationID": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		ram := raccess.ResourceAccess(p)
		acc := gqlctx.Account(ctx)
		svc := serviceFromParams(p)

		var in savedMessagesQueryInput
		if err := gqldecode.Decode(p.Args, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		ent, err := raccess.EntityInOrgForAccountID(ctx, ram, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth:             0,
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}, in.OrganizationID)
		if err == raccess.ErrNotFound {
			return nil, errors.ErrNotAuthorized(ctx, in.OrganizationID)
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		res, err := ram.SavedMessages(ctx, &threading.SavedMessagesRequest{By: &threading.SavedMessagesRequest_EntityIDs{
			EntityIDs: &threading.IDList{IDs: []string{ent.ID, in.OrganizationID}},
		}})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		sec := []*models.SavedMessageSection{
			{Title: "Your Saved Messages"},
			{Title: "Team Saved Message"},
		}

		sort.Sort(savedMessageByTitle(res.SavedMessages))
		for _, m := range res.SavedMessages {
			sm, err := transformSavedMessageToResponse(m, svc.webDomain, svc.mediaAPIDomain)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if m.OwnerEntityID == in.OrganizationID {
				sm.Shared = true
				sec[1].Messages = append(sec[1].Messages, sm)
			} else {
				sec[0].Messages = append(sec[0].Messages, sm)
			}
		}
		// Remove empty sections
		if len(sec[1].Messages) == 0 {
			sec = sec[:1]
		}
		if len(sec[0].Messages) == 0 {
			sec = sec[1:]
		}
		return sec, nil
	}),
}

func transformSavedMessageToResponse(m *threading.SavedMessage, webDomain, mediaAPIDomain string) (*models.SavedMessage, error) {
	ti, err := transformThreadItemToResponse(&threading.ThreadItem{
		ID:             m.ID,
		Timestamp:      m.Modified,
		ActorEntityID:  m.CreatorEntityID,
		Internal:       m.Internal,
		OrganizationID: m.OrganizationID,
		Item: &threading.ThreadItem_Message{
			Message: m.GetMessage(),
		},
	}, m.ID, webDomain, mediaAPIDomain)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &models.SavedMessage{
		ID:         m.ID,
		Title:      m.Title,
		ThreadItem: ti,
	}, nil
}

type savedMessageByTitle []*threading.SavedMessage

func (s savedMessageByTitle) Len() int           { return len(s) }
func (s savedMessageByTitle) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }
func (s savedMessageByTitle) Less(a, b int) bool { return s[a].Title < s[b].Title }
