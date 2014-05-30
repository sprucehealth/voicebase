package notify

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/smtp"
	"time"
)

var smtpConnectTimeout = time.Second * 5

func (n *NotificationManager) SendEmail(from, to, subject, body string) error {
	cn, err := n.smtpConnection()
	if err != nil {
		return err
	}
	defer cn.Close()
	if err := cn.Mail(from); err != nil {
		return err
	}
	if err := cn.Rcpt(to); err != nil {
		return err
	}
	wr, err := cn.Data()
	if err != nil {
		return err
	}
	defer wr.Close()
	header := http.Header{}
	header.Set("From", from)
	header.Set("To", to)
	header.Set("Subject", subject)
	if err := header.Write(wr); err != nil {
		return err
	}
	if _, err := wr.Write([]byte("\r\n")); err != nil {
		return err
	}
	if _, err := wr.Write([]byte(body)); err != nil {
		return err
	}
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
