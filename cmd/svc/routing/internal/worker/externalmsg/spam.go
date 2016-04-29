package externalmsg

import (
	"strings"

	"github.com/sprucehealth/backend/svc/excomms"
)

var spamTextPhrases = []string{
	"(WeChat Verification Code)",
	"Your TALK2 verification code is",
	"is your verification code for Instanumber",
	"Your Swytch PIN :",
	"The code is only used for removing WeChat restrictions. Do not share it with anyone.",
	"You can also tap on this link to verify your phone: v.whatsapp.com",
	"Close this message and enter the code into Facebook to confirm your phone number.",
	"[Alibaba Group]Your verification code for validation is",
	"PayPal: Your mobile number is linked to your account. To check balance, reply with BAL",
	"Your ESIAtalk number is",
	"is your Facebook confirmation code",
	"is your AOL verification code.",
	"Jelastic account activation code:",
	"Your textPlus access code is",
	"您申请注册微博，验证码",
}

func isMessageSpam(pem *excomms.PublishedExternalMessage) bool {

	if pem.Type == excomms.PublishedExternalMessage_SMS && pem.Direction == excomms.PublishedExternalMessage_INBOUND {
		text := pem.GetSMSItem().Text
		for _, phrase := range spamTextPhrases {
			if strings.Contains(text, phrase) {
				return true
			}
		}
	}
	return false
}
