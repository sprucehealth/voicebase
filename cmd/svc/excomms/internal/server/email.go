package server

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/smtpapi-go"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
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

	return sg.sgClient.Send(&sendgrid.SGMail{
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
	})
}
