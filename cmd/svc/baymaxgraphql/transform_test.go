package main

import (
	"testing"

	"github.com/sprucehealth/backend/svc/directory"
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
				Type:      directory.EntityType_INTERNAL,
				AccountID: "",
			},
			Expected: false,
		},
	}
	for _, c := range cases {
		test.Equals(t, c.Expected, entityHasPendingInvite(c.Entity))
	}
}
