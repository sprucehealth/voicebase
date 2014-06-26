package email

import (
	"github.com/sprucehealth/backend/libs/golog"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/smtp"
	"time"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

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
	SendEmail(em *Email) error
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

func (m *service) SendEmail(em *Email) error {
	if em.From == "" {
		return ErrNoSender
	}
	if em.To == "" {
		return ErrNoRecipients
	}
	if len(em.Subject) == 0 {
		return ErrEmptySubject
	}
	if len(em.BodyText) == 0 {
		return ErrEmptyBody
	}

	cn, err := m.connection()
	if err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to establish SMTP connection: %s", err)
		return err
	}
	defer cn.Close()
	if err := cn.Mail(em.From); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to issue mail command to SMTP server: %s", err)
		return err
	}
	if err := cn.Rcpt(em.To); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to issue rcpt command to SMTP server: %s", err)
		return err
	}
	wr, err := cn.Data()
	if err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to issue data command to SMTP server: %s", err)
		return err
	}
	defer wr.Close()
	header := http.Header{}
	header.Set("From", em.From)
	header.Set("To", em.To)
	header.Set("Subject", em.Subject)
	if err := header.Write(wr); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to write email header: %s", err)
		return err
	}
	if _, err := wr.Write([]byte("\r\n")); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to write email data: %s", err)
		return err
	}
	if _, err := wr.Write([]byte(em.BodyText)); err != nil {
		m.statFailed.Inc(1)
		golog.Errorf("Unable to write email body: %s", err)
		return err
	}
	m.statSent.Inc(1)
	return nil
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
