package email

import (
	"sync"

	"github.com/sprucehealth/backend/libs/mandrill"
)

type NullService struct{}

func (s NullService) Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error) {
	return nil, nil
}

type TestEmail struct {
	AccountIDs []int64
	Type       string
	Vars       map[int64][]mandrill.Var
	Msg        *mandrill.Message
}

type TestService struct {
	email []*TestEmail
	mu    sync.Mutex
}

func (m *TestService) Send(accountIDs []int64, emailType string, vars map[int64][]mandrill.Var, msg *mandrill.Message, opt Option) ([]*mandrill.SendMessageResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.email = append(m.email, &TestEmail{
		AccountIDs: accountIDs,
		Type:       emailType,
		Vars:       vars,
		Msg:        msg,
	})
	return nil, nil
}

func (m *TestService) Reset() []*TestEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	em := m.email
	m.email = nil
	return em
}

type TestMandrill struct {
	email []*TestEmail
	mu    sync.Mutex
}

func (m *TestMandrill) SendMessageTemplate(name string, content []mandrill.Var, msg *mandrill.Message, async bool) ([]*mandrill.SendMessageResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.email = append(m.email, &TestEmail{
		Type: name,
		Msg:  msg,
	})
	return nil, nil
}

func (m *TestMandrill) Reset() []*TestEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	em := m.email
	m.email = nil
	return em
}
