package mockapi

import (
	"crypto/rand"
	"encoding/hex"
	"carefront/api"
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

func (m *MockAuth) Signup(email, password string) (token string, accountId int64, err error) {
	// 
	return "", 0, nil
}

func (m *MockAuth) Login(login, password string) (token string, accountId int64, err error) {
	if account, ok := m.Accounts[login]; !ok || account.Password != password {
		return "", 0, api.ErrLoginFailed
	} else {
		tokBytes := make([]byte, 16)
		if _, err := rand.Read(tokBytes); err != nil {
			return "", 0, err
		}
		tok := hex.EncodeToString(tokBytes)
		if m.Tokens == nil {
			m.Tokens = make(map[string]int64)
		}
		m.Tokens[tok] = account.Id
		return tok, account.Id, nil
	}
}

func (m *MockAuth) Logout(token string) error {
	delete(m.Tokens, token)
	return nil
}

func (m *MockAuth) ValidateToken(token string) (valid bool, accountId int64, err error) {
	if m.Tokens != nil {
		if id, ok := m.Tokens[token]; ok {
			return true, id, nil
		}
	}
	return false, 0, nil
}
