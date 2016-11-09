package main

import (
	"context"
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
)

var scheduledMessageType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ScheduledMessage",
	Fields: graphql.Fields{
		"id":                    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"threadItem":            &graphql.Field{Type: graphql.NewNonNull(threadItemType)},
		"scheduledForTimestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})

func getScheduledMessages(ctx context.Context, ram raccess.ResourceAccessor, threadID, organizationID, webDomain, mediaAPIDomain string) ([]*models.ScheduledMessage, error) {
	resp, err := ram.ScheduledMessages(ctx, &threading.ScheduledMessagesRequest{
		LookupKey: &threading.ScheduledMessagesRequest_ThreadID{
			ThreadID: threadID,
		},
		// At the graphql layer just show pending
		Status: []threading.ScheduledMessageStatus{threading.SCHEDULED_MESSAGE_STATUS_PENDING},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	scheduledMessages, err := transformScheduledMessagesToResponse(ctx, resp.ScheduledMessages, organizationID, webDomain, mediaAPIDomain)
	if err != nil {
		return nil, errors.Trace(err)
	}
	sort.Sort(scheduledMessageByScheduledFor(scheduledMessages))
	return scheduledMessages, nil
}

func transformScheduledMessagesToResponse(ctx context.Context, ms []*threading.ScheduledMessage, organizationID, webDomain, mediaAPIDomain string) ([]*models.ScheduledMessage, error) {
	rms := make([]*models.ScheduledMessage, len(ms))
	for i, m := range ms {
		rm, err := transformScheduledMessageToResponse(ctx, m, organizationID, webDomain, mediaAPIDomain)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rms[i] = rm
	}
	return rms, nil
}

func transformScheduledMessageToResponse(ctx context.Context, m *threading.ScheduledMessage, organizationID, webDomain, mediaAPIDomain string) (*models.ScheduledMessage, error) {
	ti, err := transformThreadItemToResponse(&threading.ThreadItem{
		// Munge the ID of the thread item to not duplicate with the scheduled message but also to indicate it's a thread item
		// This is to prevent relay from getting confused
		ID:                m.ID + "_ti",
		CreatedTimestamp:  m.Created,
		ModifiedTimestamp: m.Modified,
		ActorEntityID:     m.ActorEntityID,
		Internal:          m.Internal,
		OrganizationID:    organizationID,
		Item: &threading.ThreadItem_Message{
			Message: m.GetMessage(),
		},
	}, m.ID, webDomain, mediaAPIDomain)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Mark the thread item timestamp to be the scheduled for time of the scheduled message
	ti.Timestamp = m.ScheduledFor
	return &models.ScheduledMessage{
		ID:           m.ID,
		ScheduledFor: m.ScheduledFor,
		ThreadItem:   ti,
	}, nil
}

func isScheduledMessagesEnabled() func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		if acc.Type != auth.AccountType_PROVIDER {
			return false, nil
		}
		var orgID string
		switch s := p.Source.(type) {
		case *models.Organization:
			if s == nil {
				return false, nil
			}
			orgID = s.ID
		case *models.Thread:
			if s == nil {
				return false, nil
			}
			orgID = s.OrganizationID
		default:
			return nil, errors.Errorf("Unhandled source type %T for isScheduledMessagesEnabled, returning false", s)
		}
		booleanValue, err := settings.GetBooleanValue(ctx, svc.settings, &settings.GetValuesRequest{
			NodeID: orgID,
			Keys: []*settings.ConfigKey{
				{
					Key: baymaxgraphqlsettings.ConfigKeyScheduledMessages,
				},
			},
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		return booleanValue.Value, nil
	}
}

type scheduledMessageByScheduledFor []*models.ScheduledMessage

func (s scheduledMessageByScheduledFor) Len() int      { return len(s) }
func (s scheduledMessageByScheduledFor) Swap(a, b int) { s[a], s[b] = s[b], s[a] }
func (s scheduledMessageByScheduledFor) Less(a, b int) bool {
	return s[a].ScheduledFor > s[b].ScheduledFor
}
