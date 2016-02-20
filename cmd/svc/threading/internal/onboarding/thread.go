package onboarding

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
)

func Message(step int, skip bool, webDomain, orgID string, args map[string]string) string {
	switch step {
	case 0:
		return `Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.

<a href="https://` + webDomain + `/org/` + orgID + `/settings/phone">Get your Spruce number</a>
or type "skip" to get it later`
	case 1:
		var msg string
		if skip {
			msg = `You can set up your Spruce number at any time from the settings menu. Would you like to set up your account to send and receive email through Spruce?`
		} else {
			pn, err := phone.Format(args["phoneNumber"], phone.Pretty)
			if err != nil {
				golog.Errorf("Failed to format phone number '%s': %s", args["phoneNumber"], err)
				pn = args["phoneNumber"]
			}
			msg = `Success! Your patients can now reach you at ` + pn + `. Next let’s set up you up to send and receive email through Spruce.`
		}
		msg += `

<a href="https://` + webDomain + `/org/` + orgID + `/settings/email">Set up email support</a>
or type "skip" to set it up later`
		return msg
	case 2:
		var msg string
		if skip {
			msg = `You can set up your Spruce email at any time from the settings menu. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.`
		} else {
			msg = `Great! Your patients can now reach you at ` + args["email"] + `. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.`
		}
		msg += `

<a href="https://` + webDomain + `/org/` + orgID + `/invite">Add a colleague to your organization</a>
or type "skip" to send invites later`
		return msg
	case 3:
		if skip {
			return `You can invite a colleague any time from the settings menu. Until then, you can still make internal notes on a patient conversation thread. These will only be visible to you until you add colleagues. 

You can test out internal messaging by writing a message in this conversation and tapping the lock icon before sending it.`
		}
		return `We’ve sent your invite to colleague. Once they’ve joined, you can communicate with them about care, right from a patient’s conversation thread.

To send internal messages or notes in a patient thread, simply tap the lock icon while writing a message to mark it as internal. You can test it out right here.`
	case 4:
		return `That’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the Team Spruce conversation thread and a real human will respond.`
	}
	return ""
}
