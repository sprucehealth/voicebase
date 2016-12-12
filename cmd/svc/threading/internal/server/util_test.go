package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestThreadMatchesQuery(t *testing.T) {
	now := time.Now()
	cases := map[string]struct {
		t   *models.Thread
		te  *models.ThreadEntity
		ext bool
		q   *models.Query
		m   bool
	}{
		// Matches
		"token in system title": {
			t:   &models.Thread{SystemTitle: "Joe"},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "Jo"}}}},
			m:   true,
		},
		"token in summary": {
			t:   &models.Thread{LastMessageSummary: "Joe"},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "oe"}}}},
			m:   true,
		},
		"token in external summary": {
			t:   &models.Thread{LastExternalMessageSummary: "Joe"},
			te:  nil,
			ext: true,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "oe"}}}},
			m:   true,
		},
		"has tag": {
			t:   &models.Thread{Tags: []models.Tag{{Name: "FOO"}}},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Tag{Tag: "foo"}}}},
			m:   true,
		},
		"unread no thread entity": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:   true,
		},
		"unread": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{LastViewed: ptr.Time(now.Add(-time.Second))},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:   true,
		},
		"unread reference not viewed": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{LastReferenced: &now},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}},
			m:   true,
		},
		"unread reference": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{LastReferenced: &now, LastViewed: ptr.Time(now.Add(-time.Second))},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}},
			m:   true,
		},
		"following": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{Following: true},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_FOLLOWING}}}},
			m:   true,
		},
		"patient thread": {
			t:   &models.Thread{Type: models.ThreadTypeSecureExternal},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}}},
			m:   true,
		},
		"external patient thread": {
			t:   &models.Thread{Type: models.ThreadTypeExternal},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}}},
			m:   true,
		},
		"team thread": {
			t:   &models.Thread{Type: models.ThreadTypeTeam},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_TEAM}}}},
			m:   true,
		},
		"support thread": {
			t:   &models.Thread{Type: models.ThreadTypeSupport},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_SUPPORT}}}},
			m:   true,
		},
		"setup thread": {
			t:   &models.Thread{Type: models.ThreadTypeSetup},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_SUPPORT}}}},
			m:   true,
		},
		// Negative matches
		"not token in system title": {
			t:   &models.Thread{SystemTitle: "Bob"},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Not: true, Value: &models.Expr_Token{Token: "Jo"}}}},
			m:   true,
		},
		"not including tag": {
			t:   &models.Thread{Tags: []models.Tag{{Name: "bar"}}},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Not: true, Value: &models.Expr_Token{Token: "foo"}}}},
			m:   true,
		},
		"not unread no thread entity": {
			t:   &models.Thread{MessageCount: 0},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Not: true, Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:   true,
		},
		// Non matches
		"unmatched token": {
			t:   &models.Thread{SystemTitle: "Esther"},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "Jo"}}}},
			m:   false,
		},
		"unmatched tag": {
			t:   &models.Thread{Tags: []models.Tag{{Name: "bar"}}},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Tag{Tag: "foo"}}}},
			m:   false,
		},
		"unmatched unread": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{LastViewed: &now},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:   false,
		},
		"unmatched unread reference no thread entity": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}},
			m:   false,
		},
		"unmatched following no thread entity": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_FOLLOWING}}}},
			m:   false,
		},
		"unmatched support thread": {
			t:   &models.Thread{Type: models.ThreadTypeExternal},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_SUPPORT}}}},
			m:   false,
		},
		// Negative non-matches
		"unmatched not token in system title": {
			t:   &models.Thread{SystemTitle: "Joe"},
			te:  nil,
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Not: true, Value: &models.Expr_Token{Token: "Jo"}}}},
			m:   false,
		},
		"unmatched not unread": {
			t:   &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te:  &models.ThreadEntity{LastViewed: ptr.Time(now.Add(-time.Second))},
			ext: false,
			q:   &models.Query{Expressions: []*models.Expr{{Not: true, Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:   false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			matches, err := threadMatchesQuery(c.q, c.t, c.te, c.ext)
			if err != nil {
				t.Error(err)
			}
			if matches != c.m {
				t.Errorf("Expected matching %t, got %t", c.m, matches)
			}
		})
	}
}

func TestIsUnread(t *testing.T) {
	cases := map[string]struct {
		t  *models.Thread
		te *models.ThreadEntity
		ex bool
		un bool
	}{
		"no messages": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 0), MessageCount: 0},
			te: nil,
			ex: false,
			un: false,
		},
		"no entity info": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 0), MessageCount: 1},
			te: nil,
			ex: false,
			un: true,
		},
		"not viewed": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 0), MessageCount: 1},
			te: &models.ThreadEntity{LastViewed: nil},
			ex: false,
			un: true,
		},
		"new message": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 0), MessageCount: 1},
			te: &models.ThreadEntity{LastViewed: ptr.Time(time.Unix(10, 0))},
			ex: false,
			un: true,
		},
		"read": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 0), MessageCount: 1},
			te: &models.ThreadEntity{LastViewed: ptr.Time(time.Unix(50, 0))},
			ex: false,
			un: false,
		},
		// make sure a slight variation in time under a second doesn't affect the results (truncate)
		"read truncated time": {
			t:  &models.Thread{LastMessageTimestamp: time.Unix(50, 100), MessageCount: 1},
			te: &models.ThreadEntity{LastViewed: ptr.Time(time.Unix(50, 0))},
			ex: false,
			un: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			un := isUnread(c.t, c.te, c.ex)
			if un != c.un {
				t.Fatalf("isUnread(%+v, %+v, %t) = %t. Expected %t",
					c.t, c.te, c.ex, un, c.un)
			}
		})
	}
}

func TestCreatePostMessageRequest(t *testing.T) {
	threadID, err := models.NewThreadID()
	test.OK(t, err)
	req, err := createPostMessageRequest(nil, threadID, "entity_1", false, &threading.MessagePost{
		Text:        "Foo",
		Summary:     "summary",
		Title:       "title",
		Attachments: nil, // TODO
		Source: &threading.Endpoint{
			Channel: threading.ENDPOINT_CHANNEL_SMS,
			ID:      "+11231231234",
		},
		Destinations: []*threading.Endpoint{
			{Channel: threading.ENDPOINT_CHANNEL_VOICE, ID: "+14255551212"},
		},
	})
	test.OK(t, err)
	test.Equals(t, &dal.PostMessageRequest{
		ThreadID:     threadID,
		Text:         "Foo",
		Summary:      "summary",
		Title:        "title",
		Attachments:  nil,
		FromEntityID: "entity_1",
		Source:       &models.Endpoint{Channel: models.ENDPOINT_CHANNEL_SMS, ID: "+11231231234"},
		Destinations: []*models.Endpoint{
			{Channel: models.ENDPOINT_CHANNEL_VOICE, ID: "+14255551212"},
		},
	}, req)
}
