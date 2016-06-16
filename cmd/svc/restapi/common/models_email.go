package common

import (
	"net/mail"
	"time"
)

type EmailSender struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

func (s *EmailSender) Address() *mail.Address {
	return &mail.Address{
		Name:    s.Name,
		Address: s.Email,
	}
}

type EmailTemplate struct {
	ID               int64     `json:"id"`
	Type             string    `json:"type"`
	Name             string    `json:"name"`
	SenderID         int64     `json:"sender_id"`
	SubjectTemplate  string    `json:"subject_template"`
	BodyTextTemplate string    `json:"body_text_template"`
	BodyHTMLTemplate string    `json:"body_html_template"`
	Active           bool      `json:"active"`
	Created          time.Time `json:"created"`
	Modified         time.Time `json:"modified"`
}
