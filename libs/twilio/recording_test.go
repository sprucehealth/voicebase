package twilio

import (
	"github.com/sprucehealth/backend/libs/test"

	"testing"
)

func TestParseRecordingSID(t *testing.T) {
	recordingSID, err := ParseRecordingSID(`https://api.twilio.com/2010-04-01/Accounts/AC92547608ed6ed2827fe0339b001e6166/Recordings/REf0da1dd28ab54ed1d4c192364a264366.mp3?Download=true`)
	test.OK(t, err)
	test.Equals(t, "REf0da1dd28ab54ed1d4c192364a264366", recordingSID)

	recordingSID, err = ParseRecordingSID(`https://api.twilio.com/2010-04-01/Accounts/AC92547608ed6ed2827fe0339b001e6166/Recordings/REf0da1dd28ab54ed1d4c192364a264366?Download=true`)
	test.OK(t, err)
	test.Equals(t, "REf0da1dd28ab54ed1d4c192364a264366", recordingSID)
}
