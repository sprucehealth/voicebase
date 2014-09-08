package email

import "net/mail"

type TestTemplated struct {
	To         *mail.Address
	Type       string
	TemplateID int64
	Ctx        interface{}
}

type TestService struct {
	Email     []*Email
	Templated []*TestTemplated
}

func (m *TestService) Send(e *Email) error {
	m.Email = append(m.Email, e)
	return nil
}

func (m *TestService) SendTemplate(to *mail.Address, templateID int64, ctx interface{}) error {
	m.Templated = append(m.Templated, &TestTemplated{
		To:         to,
		TemplateID: templateID,
		Ctx:        ctx,
	})
	return nil
}

func (m *TestService) SendTemplateType(to *mail.Address, typeKey string, ctx interface{}) error {
	m.Templated = append(m.Templated, &TestTemplated{
		To:   to,
		Type: typeKey,
		Ctx:  ctx,
	})
	return nil
}
