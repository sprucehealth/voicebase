package phone

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestNumber_Valid(t *testing.T) {
	testValidNumber("+17348465522", t)
	testValidNumber("+97143930033", t)
	testValidNumber("+6771234", t)
}

func testValidNumber(str string, t *testing.T) {
	_, err := ParseNumber(str)
	test.OK(t, err)
}

func TestNumber_Invalid(t *testing.T) {
	testInvalidNumber("agih", t)
	testInvalidNumber("+12068773590a", t)
	testInvalidNumber("12068773590", t)
	testInvalidNumber("+1 206 877 3590", t)
	testInvalidNumber("+1234567890123456", t)
}

func testInvalidNumber(str string, t *testing.T) {
	_, err := ParseNumber(str)
	test.Equals(t, true, err != nil)
}

func TestNumber_Marshal(t *testing.T) {
	n := Number("+12068773590")
	str, err := n.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte("+12068773590"), str)
	test.OK(t, n.UnmarshalText(str))

	n = Number("")
	str, err = n.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte(""), str)
	err = n.UnmarshalText(str)
	test.Equals(t, true, err != nil)
}

func TestNumber_Format(t *testing.T) {
	n := Number("+12068773590")

	str, err := n.Format(E164)
	test.OK(t, err)
	test.Equals(t, n.String(), str)

	str, err = n.Format(International)
	test.OK(t, err)
	test.Equals(t, "+1 206 877 3590", str)

	str, err = n.Format(National)
	test.OK(t, err)
	test.Equals(t, "206 877 3590", str)

	// only supported in the US
	n = Number("+6771234")
	str, err = n.Format(E164)
	test.Equals(t, true, err != nil)
}

func BenchmarkFormat(b *testing.B) {
	n := Number("+12068773590")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		n.Format(International)
		n.Format(National)
		n.Format(E164)
	}
}
