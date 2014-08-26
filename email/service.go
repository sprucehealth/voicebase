package email

import (
	"crypto/tls"
	"errors"
	"net"
	"net/mail"
	"net/smtp"
	"time"

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
	Send(em *Email) error
}

type service struct {
	config     *Config
	statSent   metrics.Counter
	statFailed metrics.Counter
}

func NewService(config *Config, metricsRegistry metrics.Registry) *service {
	if config.SMTPConnectTimeout == 0 {
		config.SMTPConnectTimeout = defaultConnectTimeout
	}
	m := &service{
		config:     config,
		statSent:   metrics.NewCounter(),
		statFailed: metrics.NewCounter(),
	}
	metricsRegistry.Add("sent", m.statSent)
	metricsRegistry.Add("failed", m.statSent)
	return m
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
