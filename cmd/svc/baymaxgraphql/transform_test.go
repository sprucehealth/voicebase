package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/threading"
)

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
			Thread:   &threading.Thread{Type: threading.THREAD_TYPE_TEAM},
			Expected: "This is the beginning of your team conversation.\nSend a message to get things started.",
		},
	}
	for _, c := range cases {
		test.Equals(t, c.Expected, threadEmptyStateTextMarkup(c.Ctx, c.RA, c.Thread, c.Acc))
	}
}
