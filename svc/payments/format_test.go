package payments

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestFormatAmount(t *testing.T) {

	amount := FormatAmount(4600, "USD")
	test.Equals(t, "$46", amount)
	amount = FormatAmount(400000, "USD")
	test.Equals(t, "$4,000", amount)
	amount = FormatAmount(460, "USD")
	test.Equals(t, "$4.60", amount)
	amount = FormatAmount(461, "USD")
	test.Equals(t, "$4.61", amount)
	amount = FormatAmount(400061, "USD")
	test.Equals(t, "$4,000.61", amount)
	amount = FormatAmount(4000123461, "USD")
	test.Equals(t, "$40,001,234.61", amount)
	amount = FormatAmount(469, "USD")
	test.Equals(t, "$4.69", amount)

}
