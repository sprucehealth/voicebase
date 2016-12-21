package externalmsg

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/excomms"
)

func TestIsMessageSpam(t *testing.T) {
	test.Equals(t, true, isMessageSpam(&excomms.PublishedExternalMessage{
		Item: &excomms.PublishedExternalMessage_SMSItem{
			SMSItem: &excomms.SMSItem{
				Text: "WeChat verification code (7152) is only used to change linking. Forwarding the code to others may compromise your account.",
			},
		},
	}))
}
