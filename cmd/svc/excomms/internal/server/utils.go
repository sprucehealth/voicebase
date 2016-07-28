package server

import (
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/validate"
)

// domainFromEmail returns the "y" of x@y.z or x@subdomain.y.z.
func domainFromEmail(address string) (string, error) {
	if !validate.Email(address) {
		return "", errors.Errorf("invalid email %s", address)
	}

	idx1 := strings.LastIndex(address, ".")
	idx2 := strings.LastIndex(address[:idx1], ".")
	if idx2 == -1 {
		idx2 = strings.Index(address[:idx1], "@")
	}

	return address[idx2+1 : idx1], nil
}
