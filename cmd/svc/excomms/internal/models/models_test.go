package models

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/test"
)

func TestProxyPhoneNumber_Sort(t *testing.T) {
	p := []*ProxyPhoneNumber{
		{
			PhoneNumber: phone.Number("+12068773590"),
		},
		{
			PhoneNumber: phone.Number("+12068773591"),
		},
		{
			PhoneNumber: phone.Number("+12068773592"),
		},
	}

	sort.Sort(ByLastReservedProxyPhoneNumbers(p))

	test.Equals(t, "+12068773590", p[0].PhoneNumber.String())
	test.Equals(t, "+12068773591", p[1].PhoneNumber.String())
	test.Equals(t, "+12068773592", p[2].PhoneNumber.String())

}
