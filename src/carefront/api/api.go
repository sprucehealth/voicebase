package api

import "errors"

var ErrLoginFailed = errors.New("api: login failed")

type Auth interface {
	Login(login, password string) (token string, err error)
	Logout(token string) error
	ValidateToken(token string) (valid bool, accountId int64, err error)
}
