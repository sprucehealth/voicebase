package server

import (
	"fmt"
	"net/http"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
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
		resp, err := http.Get(url)
		if err != nil {
			return errors.Trace(err)
		}
		defer resp.Body.Close()
		if err := sgMail.AddAttachment(fmt.Sprintf("media_attachment_%d", i), resp.Body); err != nil {
			return errors.Trace(err)
		}
	}
	return sg.sgClient.Send(sgMail)
}
