package email

type TestService struct {
	Email []*Email
}

func (m *TestService) Send(e *Email) error {
	m.Email = append(m.Email, e)
	return nil
}
