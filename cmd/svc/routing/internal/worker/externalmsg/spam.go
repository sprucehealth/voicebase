package externalmsg

import (
	"strings"

	"github.com/sprucehealth/backend/svc/excomms"
)

var spamTextPhrases = []string{
	"WeChat Verification Code",
	"Your TALK2 verification code is",
	"is your verification code for Instanumber",
	"Your Swytch PIN :",
	"The code is only used for removing WeChat restrictions. Do not share it with anyone.",
	"You can also tap on this link to verify your phone: v.whatsapp.com",
	"[Alibaba Group]Your verification code for validation is",
	"Your ESIAtalk number is",
	"is your AOL verification code.",
	"Jelastic account activation code:",
	"Your textPlus access code is",
	"您申请注册微博，验证码",
	"Your Virtual SIM  verification code",
}

func isMessageSpam(pem *excomms.PublishedExternalMessage) bool {
	if pem.Type == excomms.PublishedExternalMessage_SMS && pem.Direction == excomms.PublishedExternalMessage_INBOUND {
		text := pem.GetSMSItem().Text
		for _, phrase := range spamTextPhrases {
			if strings.Contains(strings.ToLower(text), strings.ToLower(phrase)) {
				return true
			}
		}
	}
	return false
}
