package server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

type EmailClient interface {
	SendMessage(em *models.EmailMessage) error
}

type sg struct {
	apiKey string
}

func NewSendgridClient(apiKey string) EmailClient {
	return &sg{
		apiKey: apiKey,
	}
}

func (sg *sg) SendMessage(em *models.EmailMessage) error {
	body := em.Body
	if body == "" {
		body = "\t"
	}
	if em.Subject == "" {
		em.Subject = " "
	}
	m := mail.NewV3MailInit(
		mail.NewEmail(em.FromName, em.FromEmail), em.Subject,
		mail.NewEmail(em.ToName, em.ToEmail),
		mail.NewContent("text/plain", body))
	if em.TemplateID != "" {
		m.TemplateID = em.TemplateID
		m.Content = append(m.Content, mail.NewContent("text/html", html.EscapeString(body)))
		if len(em.TemplateSubstitutions) != 0 {
			subs := make(map[string]string, len(em.TemplateSubstitutions))
			for _, s := range em.TemplateSubstitutions {
				subs[s.Key] = s.Value
			}
			m.Personalizations[0].Substitutions = subs
		}
	}
	m.Headers = map[string]string{"X-Message-ID": em.ID}

	// Stream in any media attachments
	for i, url := range em.MediaURLs {
		resp, err := http.Get(url)
		if err != nil {
			return errors.Trace(err)
		}
		defer resp.Body.Close()

		content, err := encodeAttachment(resp.Body)
		if err != nil {
			return errors.Trace(err)
		}

		name := fmt.Sprintf("media_attachment_%d%s", i, imageExtensionFromHeader(resp.Header))

		m.Attachments = append(m.Attachments, &mail.Attachment{
			Type:     resp.Header.Get("Content-Type"),
			Content:  content,
			Filename: name,
			Name:     name,
		})
	}

	req := sendgrid.GetRequest(sg.apiKey, "/v3/mail/send", "https://api.sendgrid.com")
	req.Method = "POST"
	req.Body = mail.GetRequestBody(m)
	_, err := sendgrid.API(req)
	return errors.Trace(err)
}

func encodeAttachment(r io.Reader) (string, error) {
	b := &bytes.Buffer{}
	wc := base64.NewEncoder(base64.StdEncoding, b)
	defer wc.Close()
	_, err := io.Copy(wc, r)
	if err != nil {
		return "", errors.Trace(err)
	}
	return b.String(), nil
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
