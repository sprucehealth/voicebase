package email

import (
	"bytes"
	"crypto/tls"
	"errors"
	htmltemplate "html/template"
	"net"
	"net/mail"
	"net/smtp"
	texttemplate "text/template"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/jordan-wright/email"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

type Email email.Email

var (
	ErrEmptyBody    = errors.New("email: empty body")
	ErrEmptySubject = errors.New("email: empty subject")
	ErrNoRecipients = errors.New("email: no recipient")
	ErrNoSender     = errors.New("email: no sender")
)

var defaultConnectTimeout = time.Second * 5

type Config struct {
	SMTPAddress        string `long:"smtp_host" description:"SMTP host:port"`
	SMTPUsername       string `long:"smtp_username" description:"Username for SMTP server"`
	SMTPPassword       string `long:"smtp_password" description:"Password for SMTP server"`
	SMTPConnectTimeout time.Duration
}

type Service interface {
	Send(*Email) error
	SendTemplate(to *mail.Address, templateID int64, ctx interface{}) error
	SendTemplateType(to *mail.Address, typeKey string, ctx interface{}) error
}

type service struct {
	dataAPI    api.DataAPI
	config     *Config
	statSent   *metrics.Counter
	statFailed *metrics.Counter
}

func NewService(dataAPI api.DataAPI, config *Config, metricsRegistry metrics.Registry) Service {
	if config.SMTPConnectTimeout == 0 {
		config.SMTPConnectTimeout = defaultConnectTimeout
	}
	m := &service{
		dataAPI:    dataAPI,
		config:     config,
		statSent:   metrics.NewCounter(),
		statFailed: metrics.NewCounter(),
	}
	metricsRegistry.Add("sent", m.statSent)
	metricsRegistry.Add("failed", m.statSent)
	return m
}

func (m *service) SendTemplateType(to *mail.Address, typeKey string, ctx interface{}) error {
	tmpl, err := m.dataAPI.GetActiveEmailTemplateForType(typeKey)
	if err == api.NoRowsError {
		golog.Errorf("No active template fo remail type %s", typeKey)
		return err
	} else if err != nil {
		return err
	}
	return m.sendTemplate(to, tmpl, ctx)
}

func (m *service) SendTemplate(to *mail.Address, templateID int64, ctx interface{}) error {
	tmpl, err := m.dataAPI.GetEmailTemplate(templateID)
	if err == api.NoRowsError {
		golog.Errorf("Template %d not found", templateID)
		return err
	} else if err != nil {
		return err
	}
	return m.sendTemplate(to, tmpl, ctx)
}

func (m *service) sendTemplate(to *mail.Address, tmpl *common.EmailTemplate, ctx interface{}) error {
	log := golog.Context("email_type", tmpl.Type, "template_id", tmpl.ID)

	subjectTmpl, err := texttemplate.New("").Parse(tmpl.SubjectTemplate)
	if err != nil {
		log.Errorf("Failed to parse email subject template: %s", err.Error())
		return err
	}
	htmlTmpl, err := htmltemplate.New("").Parse(tmpl.BodyHTMLTemplate)
	if err != nil {
		log.Errorf("Failed to parse email HTML body template: %s", err.Error())
		return err
	}
	textTmpl, err := texttemplate.New("").Parse(tmpl.BodyTextTemplate)
	if err != nil {
		log.Errorf("Failed to parse email text body template: %s", err.Error())
		return err
	}

	sender, err := m.dataAPI.GetEmailSender(tmpl.SenderID)
	if err != nil {
		return err
	}

	em := &Email{
		To:   []string{to.String()},
		From: sender.Address().String(),
	}

	b := &bytes.Buffer{}
	if err := subjectTmpl.Execute(b, ctx); err != nil {
		log.Errorf("Failed to render email subject template: %s", err.Error())
		return err
	}
	em.Subject = b.String()

	b.Reset()
	if err := htmlTmpl.Execute(b, ctx); err != nil {
		log.Errorf("Failed to render email HTML body template: %s", err.Error())
		return err
	}
	em.HTML = b.Bytes()

	b = &bytes.Buffer{}
	if err := textTmpl.Execute(b, ctx); err != nil {
		log.Errorf("Failed to render email text body template: %s", err.Error())
		return err
	}
	em.Text = b.Bytes()

	return m.Send(em)
}

func (m *service) Send(em *Email) error {
	e := (*email.Email)(em)

	if e.From == "" {
		return ErrNoSender
	}
	if len(e.Subject) == 0 {
		return ErrEmptySubject
	}
	if len(e.HTML) == 0 && len(e.Text) == 0 {
		return ErrEmptyBody
	}

	to := make([]string, 0, len(e.To)+len(e.Cc)+len(e.Bcc))
	to = append(append(append(to, e.To...), e.Cc...), e.Bcc...)
	for i := 0; i < len(to); i++ {
		addr, err := mail.ParseAddress(to[i])
		if err != nil {
			return err
		}
		to[i] = addr.Address
	}
	if len(to) == 0 {
		return ErrNoRecipients
	}

	from, err := mail.ParseAddress(e.From)
	if err != nil {
		return err
	}
	raw, err := e.Bytes()
	if err != nil {
		return err
	}

	if err := m.send(from.Address, to, raw); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf(err.Error())
		return err
	}

	m.statSent.Inc(1)
	return nil
}

func (m *service) send(from string, to []string, rawBody []byte) error {
	cn, err := m.connection()
	if err != nil {
		return err
	}
	defer cn.Close()

	if err := cn.Mail(from); err != nil {
		return err
	}
	for _, t := range to {
		if err := cn.Rcpt(t); err != nil {
			return err
		}
	}
	wr, err := cn.Data()
	if err != nil {
		return err
	}
	if _, err := wr.Write(rawBody); err != nil {
		wr.Close()
		return err
	}
	return wr.Close()
}

func (m *service) connection() (*smtp.Client, error) {
	host, _, _ := net.SplitHostPort(m.config.SMTPAddress)
	c, err := net.DialTimeout("tcp", m.config.SMTPAddress, m.config.SMTPConnectTimeout)
	if err != nil {
		return nil, errors.New("failed to connect to SMTP server: " + err.Error())
	}
	cn, err := smtp.NewClient(c, host)
	if err != nil {
		return nil, errors.New("failed to create SMTP client: " + err.Error())
	}
	if err := cn.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return nil, errors.New("failed to StartTLS with SMTP server: " + err.Error())
	}
	if err := cn.Auth(smtp.PlainAuth("", m.config.SMTPUsername, m.config.SMTPPassword, host)); err != nil {
		return nil, errors.New("smtp auth failed: " + err.Error())
	}
	return cn, err
}
