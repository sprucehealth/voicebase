package main

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
)

func TestEntityHasPendingInvite(t *testing.T) {
	cases := []struct {
		Entity   *directory.Entity
		Expected bool
	}{
		// Patients without accounts should have pending invites
		{
			Entity: &directory.Entity{
				Type:      directory.EntityType_PATIENT,
				AccountID: "",
			},
			Expected: true,
		},
		// Non patients without accounts should not have pending invite
		{
			Entity: &directory.Entity{
				Type:      directory.EntityType_INTERNAL,
				AccountID: "",
			},
			Expected: false,
		},
		// Patients with accounts should not have pending invite
		{
			Entity: &directory.Entity{
				Type:      directory.EntityType_PATIENT,
				AccountID: "account_123456",
			},
			Expected: false,
		},
	}
	for _, c := range cases {
		test.Equals(t, c.Expected, entityHasPendingInvite(c.Entity))
	}
}

func TestThreadEmptyStateMarkup(t *testing.T) {
	cases := map[string]struct {
		Ctx      context.Context
		RA       raccess.ResourceAccessor
		Thread   *threading.Thread
		Acc      *auth.Account
		Expected string
	}{
		// Empty threads always have no ESTM
		"EmptyThreads": {
			Thread:   &threading.Thread{MessageCount: 1},
			Expected: "",
		},
		// Team threads always have the same ESTM
		"TeamThreads": {
			Thread:   &threading.Thread{Type: threading.ThreadType_TEAM},
			Expected: "This is the beginning of your team conversation.\nSend a message to get things started.",
		},
	}
	for _, c := range cases {
		test.Equals(t, c.Expected, threadEmptyStateTextMarkup(c.Ctx, c.RA, c.Thread, c.Acc))
	}
}
