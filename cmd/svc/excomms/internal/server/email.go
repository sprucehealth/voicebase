package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type EmailClient interface {
	SendMessage(em *models.EmailMessage) error
}

type sg struct {
	sgClient *sendgrid.SGClient
}

func NewSendgridClient(sendgridAPI string) EmailClient {
	return &sg{
		sgClient: sendgrid.NewSendGridClientWithApiKey(sendgridAPI),
	}
}

func (sg *sg) SendMessage(em *models.EmailMessage) error {
	sgMail := &sendgrid.SGMail{
		To:       []string{em.ToEmail},
		ToName:   []string{em.ToName},
		Subject:  em.Subject,
		Text:     em.Body,
		From:     em.FromEmail,
		FromName: em.FromName,
		SMTPAPIHeader: smtpapi.SMTPAPIHeader{
			UniqueArgs: map[string]string{
				"x_message_id": em.ID,
			},
		},
	}
	if sgMail.Text == "" {
		sgMail.Text = "\t"
	}

	// Stream in any media attachments
	for i, url := range em.MediaURLs {
		hResp, err := http.Head(url)
		if err != nil {
			return errors.Trace(err)
		}
		resp, err := http.Get(url)
		if err != nil {
			return errors.Trace(err)
		}
		defer resp.Body.Close()
		if err := sgMail.AddAttachment(fmt.Sprintf("media_attachment_%d"+imageExtensionFromHeader(hResp.Header), i), resp.Body); err != nil {
			return errors.Trace(err)
		}
	}
	return sg.sgClient.Send(sgMail)
}

// TODO: Make this more robust and libify
func imageExtensionFromHeader(header http.Header) string {
	ct := header.Get("Content-Type")
	cts := strings.Split(ct, "/")
	if len(cts) != 2 {
		golog.Errorf("Unknown content type for image extension selection %s", ct)
		return ""
	}
	if cts[0] != "image" {
		golog.Errorf("Non image type for image extension selection %s", ct)
		return ""
	}
	return "." + cts[1]
}
