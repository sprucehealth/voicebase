package service

import (
	"testing"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/test"
)

func TestExternalIDToAccountIDTransformation(t *testing.T) {
	externalIDs := []*directory.ExternalID{
		&directory.ExternalID{
			ID: "account:account:215610700746457088",
		},
		&directory.ExternalID{
			ID: "account:account:215610700746457090",
		},
		&directory.ExternalID{
			ID: "other:1235123423522",
		},
	}
	accountIDs := accountIDsFromExternalIDs(externalIDs)
	test.Equals(t, []string{"account:215610700746457088", "account:215610700746457090"}, accountIDs)
}
