package server

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
)

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
