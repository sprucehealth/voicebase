package email

import (
	"sync"

	"net/mail"
)

type TestTemplated struct {
	To         *mail.Address
	Type       string
	TemplateID int64
	Ctx        interface{}
}

type TestService struct {
	email     []*Email
	templated []*TestTemplated
	mu        sync.Mutex
}

func (m *TestService) Send(e *Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.email = append(m.email, e)
	return nil
}

func (m *TestService) SendTemplate(to *mail.Address, templateID int64, ctx interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templated = append(m.templated, &TestTemplated{
		To:         to,
		TemplateID: templateID,
		Ctx:        ctx,
	})
	return nil
}

func (m *TestService) SendTemplateType(to *mail.Address, typeKey string, ctx interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templated = append(m.templated, &TestTemplated{
		To:   to,
		Type: typeKey,
		Ctx:  ctx,
	})
	return nil
}

func (m *TestService) Reset() ([]*Email, []*TestTemplated) {
	m.mu.Lock()
	defer m.mu.Unlock()
	emails, templated := m.email, m.templated
	m.email, m.templated = nil, nil
	return emails, templated
}
