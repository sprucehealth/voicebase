package sendgrid

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
)

func ParamsFromRequest(r *http.Request) (*rawmsg.SendGridIncomingEmail, error) {
	sgi := &rawmsg.SendGridIncomingEmail{
		Headers:      r.FormValue("headers"),
		Text:         r.FormValue("text"),
		HTML:         r.FormValue("html"),
		Sender:       r.FormValue("from"),
		Recipient:    r.FormValue("to"),
		CC:           r.FormValue("cc"),
		Subject:      r.FormValue("subject"),
		DKIM:         r.FormValue("dkim"),
		SPF:          r.FormValue("spf"),
		SMTPEnvelope: r.FormValue("envelope"),
		Charsets:     r.FormValue("charsets"),
		SpamScore:    r.FormValue("spam_score"),
		SpamReport:   r.FormValue("spam_report"),
	}

	if r.FormValue("attachments") != "" {
		numAttachments, err := strconv.Atoi(r.FormValue("attachments"))
		if err != nil {
			return nil, err
		}
		sgi.NumAttachments = uint32(numAttachments)
	}

	//TODO: Parse out attachments

	return sgi, nil
}
