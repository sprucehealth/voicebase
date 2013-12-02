package mockapi

import (
	"carefront/common"
	"carefront/thriftapi"
)

type MockAccount struct {
	Id       int64
	Login    string
	Password string
}

type MockAuth struct {
	Accounts map[string]MockAccount
	Tokens   map[string]int64
}

var (
	IdCounter = 1
)

func (m *MockAuth) Signup(email, password string) (*thriftapi.AuthResponse, error) {
	if a, ok := m.Accounts[email]; ok {
		return nil, &thriftapi.LoginAlreadyExists{AccountId: a.Id}
	}
	tok, err := common.GenerateToken()
	if err != nil {
		return nil, err
	}
	if m.Tokens == nil {
		m.Tokens = make(map[string]int64)
	}
	IdCounter += 1
	m.Accounts[email] = MockAccount{int64(IdCounter), email, password}
	m.Tokens[tok] = int64(IdCounter)
	return &thriftapi.AuthResponse{Token: tok, AccountId: int64(IdCounter)}, nil
}

func (m *MockAuth) Login(login, password string) (*thriftapi.AuthResponse, error) {
	if account, ok := m.Accounts[login]; !ok || account.Password != password {
		return nil, &thriftapi.NoSuchLogin{}
	} else {
		tok, err := common.GenerateToken()
		if err != nil {
			return nil, err
		}
		if m.Tokens == nil {
			m.Tokens = make(map[string]int64)
		}
		m.Tokens[tok] = account.Id
		return &thriftapi.AuthResponse{Token: tok, AccountId: account.Id}, nil
	}
}

func (m *MockAuth) Logout(token string) error {
	delete(m.Tokens, token)
	return nil
}

func (m *MockAuth) ValidateToken(token string) (*thriftapi.TokenValidationResponse, error) {
	if m.Tokens != nil {
		if id, ok := m.Tokens[token]; ok {
			return &thriftapi.TokenValidationResponse{IsValid: true, AccountId: &id}, nil
		}
	}
	return &thriftapi.TokenValidationResponse{IsValid: false}, nil
}
