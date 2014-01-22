package mockapi

import (
	"strings"

	"carefront/common"
	"carefront/thrift/api"
)

type MockAccount struct {
	Id       int64
	Login    string
	Password string
}

type MockAuth struct {
	Accounts map[string]*MockAccount
	Tokens   map[string]int64
}

var (
	IdCounter = 1
)

func (m *MockAuth) SignUp(email, password string) (*api.AuthResponse, error) {
	email = strings.ToLower(email)
	if a, ok := m.Accounts[email]; ok {
		return nil, &api.LoginAlreadyExists{AccountId: a.Id}
	}
	tok, err := common.GenerateToken()
	if err != nil {
		return nil, err
	}
	if m.Tokens == nil {
		m.Tokens = make(map[string]int64)
	}
	IdCounter += 1
	m.Accounts[email] = &MockAccount{Id: int64(IdCounter), Login: email, Password: password}
	m.Tokens[tok] = int64(IdCounter)
	return &api.AuthResponse{Token: tok, AccountId: int64(IdCounter)}, nil
}

func (m *MockAuth) LogIn(login, password string) (*api.AuthResponse, error) {
	login = strings.ToLower(login)
	if account, ok := m.Accounts[login]; !ok || account.Password != password {
		return nil, &api.NoSuchLogin{}
	} else {
		tok, err := common.GenerateToken()
		if err != nil {
			return nil, err
		}
		if m.Tokens == nil {
			m.Tokens = make(map[string]int64)
		}
		m.Tokens[tok] = account.Id
		return &api.AuthResponse{Token: tok, AccountId: account.Id}, nil
	}
}

func (m *MockAuth) LogOut(token string) error {
	delete(m.Tokens, token)
	return nil
}

func (m *MockAuth) ValidateToken(token string) (*api.TokenValidationResponse, error) {
	if m.Tokens != nil {
		if id, ok := m.Tokens[token]; ok {
			return &api.TokenValidationResponse{IsValid: true, AccountId: &id}, nil
		}
	}
	return &api.TokenValidationResponse{IsValid: false}, nil
}

func (m *MockAuth) SetPassword(accountId int64, password string) error {
	if password == "" {
		return &api.InvalidPassword{}
	}
	// Log out any existing tokens
	for tok, aid := range m.Tokens {
		if aid == accountId {
			delete(m.Tokens, tok)
		}
	}
	for _, act := range m.Accounts {
		if act.Id == accountId {
			act.Password = password
			return nil
		}
	}
	return &api.NoSuchAccount{}
}
