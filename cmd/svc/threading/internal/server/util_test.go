package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
)

func TestThreadMatchesQuery(t *testing.T) {
	now := time.Now()
	cases := map[string]struct {
		t  *models.Thread
		te *models.ThreadEntity
		ee bool
		q  *models.Query
		m  bool
	}{
		// Matches
		"token in system title": {
			t:  &models.Thread{SystemTitle: "Joe"},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "Jo"}}}},
			m:  true,
		},
		"token in summary": {
			t:  &models.Thread{LastMessageSummary: "Joe"},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "oe"}}}},
			m:  true,
		},
		"unread no thread entity": {
			t:  &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:  true,
		},
		"unread": {
			t:  &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te: &models.ThreadEntity{LastViewed: ptr.Time(now.Add(-time.Second))},
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:  true,
		},
		"unread reference not viewed": {
			t:  &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te: &models.ThreadEntity{LastReferenced: &now},
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}},
			m:  true,
		},
		"unread reference": {
			t:  &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te: &models.ThreadEntity{LastReferenced: &now, LastViewed: ptr.Time(now.Add(-time.Second))},
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}}}},
			m:  true,
		},
		"patient thread": {
			t:  &models.Thread{Type: models.ThreadTypeSecureExternal},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}}},
			m:  true,
		},
		"external patient thread": {
			t:  &models.Thread{Type: models.ThreadTypeExternal},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}}}},
			m:  true,
		},
		"team thread": {
			t:  &models.Thread{Type: models.ThreadTypeTeam},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_TEAM}}}},
			m:  true,
		},
		// Non matches
		"unmatched token": {
			t:  &models.Thread{SystemTitle: "Esther"},
			te: nil,
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Token{Token: "Jo"}}}},
			m:  false,
		},
		"unmatched unread": {
			t:  &models.Thread{MessageCount: 1, LastMessageTimestamp: now},
			te: &models.ThreadEntity{LastViewed: &now},
			ee: false,
			q:  &models.Query{Expressions: []*models.Expr{{Value: &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}}}},
			m:  false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			matches, err := threadMatchesQuery(c.q, c.t, c.te, c.ee)
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
