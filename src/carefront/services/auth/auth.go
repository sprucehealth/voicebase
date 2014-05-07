package auth

import (
	"carefront/libs/golog"
	"database/sql"
	"strings"
	"time"

	"carefront/common"
	"carefront/thrift/api"
)

type AuthService struct {
	ExpireDuration time.Duration
	RenewDuration  time.Duration // When validation, if the time left on the token is less than this duration than the token is extended
	DB             *sql.DB
	Hasher         PasswordHasher
}

func (m *AuthService) SignUp(email, password string) (*api.AuthResponse, error) {
	if password == "" {
		return nil, &api.InvalidPassword{}
	}
	email = strings.ToLower(email)

	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.DB.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return nil, &api.LoginAlreadyExists{AccountId: id}
	} else if err != nil && err != sql.ErrNoRows {
		golog.Errorf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		golog.Errorf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		golog.Errorf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password) VALUES (?, ?)", email, string(hashedPassword))
	if err != nil {
		tx.Rollback()
		golog.Errorf("services/auth: INSERT account failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	tok, err := common.GenerateToken()
	if err != nil {
		tx.Rollback()
		golog.Errorf("services/auth: GenerateToken failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		golog.Errorf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// store token in Token Database
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", tok, lastId, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		golog.Errorf("services/auth: INSERT auth_token failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	tx.Commit()
	return &api.AuthResponse{Token: tok, AccountId: lastId}, nil
}

func (m *AuthService) LogIn(email, password string) (*api.AuthResponse, error) {
	email = strings.ToLower(email)

	var accountId int64
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.DB.QueryRow("SELECT id, password FROM account WHERE email = ?", email).Scan(&accountId, &hashedPassword); err == sql.ErrNoRows {
		return nil, &api.NoSuchLogin{}
	} else if err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := m.Hasher.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, &api.InvalidPassword{AccountId: accountId}
	}

	token, err := common.GenerateToken()
	if err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// delete any existing token and create a new one
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		return nil, &api.InternalServerError{Message: err.Error()}
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("DELETE FROM auth_token WHERE account_id = ?", accountId)
	if err != nil {
		tx.Rollback()
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// insert new token
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", token, accountId, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return nil, &api.InternalServerError{Message: err.Error()}
	}
	tx.Commit()

	return &api.AuthResponse{Token: token, AccountId: accountId}, nil
}

func (m *AuthService) LogOut(token string) error {
	// delete the token from the database to invalidate
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE token = ?", token); err != nil {
		return &api.InternalServerError{Message: err.Error()}
	}
	return nil
}

func (m *AuthService) ValidateToken(token string) (*api.TokenValidationResponse, error) {
	var accountId int64
	var expires *time.Time
	if err := m.DB.QueryRow("SELECT account_id, expires FROM auth_token WHERE token = ?", token).Scan(&accountId, &expires); err == sql.ErrNoRows {
		return &api.TokenValidationResponse{IsValid: false, Reason: "token not found"}, nil
	} else if err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	now := time.Now().UTC()

	// if the token exists, check the expiration to ensure that it is valid
	left := (*expires).Sub(now)
	reason := ""
	if left <= 0 {
		golog.Infof("Current time %s is after expiration time %s", now.String(), expires.String())
		reason = "token expired"
	} else if m.RenewDuration > 0 && left < m.RenewDuration {
		if _, err := m.DB.Exec("UPDATE auth_token SET expires = ? WHERE token = ?", now.Add(m.ExpireDuration), token); err != nil {
			golog.Errorf("services/auth: failed to extend token expiration: %s", err.Error())
			// Don't return an error response because this doesn't prevent anything else from working
		}
	}
	return &api.TokenValidationResponse{IsValid: left > 0, AccountId: &accountId, Reason: reason}, nil
}

func (m *AuthService) SetPassword(accountId int64, password string) error {
	if password == "" {
		return &api.InvalidPassword{}
	}
	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return &api.InternalServerError{Message: err.Error()}
	}
	if res, err := m.DB.Exec("UPDATE account SET password = ? WHERE id = ?", string(hashedPassword), accountId); err != nil {
		return &api.InternalServerError{Message: err.Error()}
	} else if n, err := res.RowsAffected(); err != nil {
		return &api.InternalServerError{Message: err.Error()}
	} else if n == 0 {
		return &api.NoSuchAccount{}
	}
	// Log out any existing tokens for the account
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE account_id = ?", accountId); err != nil {
		return &api.InternalServerError{Message: err.Error()}
	}
	return nil
}
