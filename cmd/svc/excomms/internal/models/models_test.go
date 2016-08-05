package models

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/test"
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

	sort.Sort(ByExpiresProxyPhoneNumbers(p))

	test.Equals(t, "+12068773590", p[0].PhoneNumber.String())
	test.Equals(t, "+12068773591", p[1].PhoneNumber.String())
	test.Equals(t, "+12068773592", p[2].PhoneNumber.String())

}

func TestBlockedNumber(t *testing.T) {
	b := BlockedNumbers(nil)
	test.Equals(t, false, b.Includes(phone.Number("+12222222222")))

	pn := phone.Number("+12222222222")
	b = BlockedNumbers([]phone.Number{pn})
	test.Equals(t, true, b.Includes(pn))

	pn2 := phone.Number("+13333333333")
	test.Equals(t, false, b.Includes(pn2))

	b = BlockedNumbers([]phone.Number{pn, pn2})
	test.Equals(t, true, b.Includes(pn))
	test.Equals(t, true, b.Includes(pn2))

}
