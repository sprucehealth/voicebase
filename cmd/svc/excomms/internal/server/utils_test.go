package server

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestDomainFromEmail(t *testing.T) {
	domain, err := domainFromEmail("from@example.com")
	test.OK(t, err)
	test.Equals(t, "example", domain)

	domain, err = domainFromEmail("from@subdomain.example.com")
	test.OK(t, err)
	test.Equals(t, "example", domain)

}
