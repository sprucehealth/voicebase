package notify

import (
	"carefront/libs/golog"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/smtp"
	"time"
)

var smtpConnectTimeout = time.Second * 5

func (n *NotificationManager) SendEmail(from, to, subject, body string) error {
	go func() {
		cn, err := n.smtpConnection()
		if err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to establish smtp connection: %s", err)
			return
		}
		defer cn.Close()
		if err := cn.Mail(from); err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to issue mail command to server: %s", err)
			return
		}
		if err := cn.Rcpt(to); err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to issue rcpt command to server: %s", err)
			return
		}
		wr, err := cn.Data()
		if err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to issue data command to server: %s", err)
			return
		}
		defer wr.Close()
		header := http.Header{}
		header.Set("From", from)
		header.Set("To", to)
		header.Set("Subject", subject)
		if err := header.Write(wr); err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to write header: %s", err)
			return
		}
		if _, err := wr.Write([]byte("\r\n")); err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to write data : %s", err)
			return
		}
		if _, err := wr.Write([]byte(body)); err != nil {
			n.statEmailFailed.Inc(1)
			golog.Errorf("Unable to write body of email: %s", err)
			return
		}
		n.statEmailSent.Inc(1)
	}()
	return nil
}

func (n *NotificationManager) smtpConnection() (*smtp.Client, error) {
	host, _, _ := net.SplitHostPort(n.smtpAddress)
	c, err := net.DialTimeout("tcp", n.smtpAddress, smtpConnectTimeout)
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
	if err := cn.Auth(smtp.PlainAuth("", n.smtpUsername, n.smtpPassword, host)); err != nil {
		return nil, errors.New("smtp auth failed: " + err.Error())
	}
	return cn, err
}
