package api

import (
	"carefront/common"
	"carefront/libs/golog"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	InvalidPassword    = errors.New("api: invalid password")
	InvalidRoleType    = errors.New("api: invalid role type")
	LoginAlreadyExists = errors.New("api: login already exists")
	LoginDoesNotExist  = errors.New("api: login does not exist")
)

type AuthResponse struct {
	Token     string `json:"token"`
	AccountId int64  `json:"account_id"`
}

type TokenValidationResponse struct {
	IsValid   bool   `json:"is_valid"`
	AccountId *int64 `json:"account_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Auth struct {
	ExpireDuration time.Duration
	RenewDuration  time.Duration // When validation, if the time left on the token is less than this duration than the token is extended
	DB             *sql.DB
	Hasher         PasswordHasher
}

func (m *Auth) SignUp(email, password, roleType string) (*AuthResponse, error) {
	if password == "" {
		return nil, InvalidPassword
	}
	email = strings.ToLower(email)

	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.DB.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return nil, LoginAlreadyExists
	} else if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return nil, err
	}

	var roleTypeId int64
	if err := m.DB.QueryRow("SELECT id from role_type where role_type_tag = ?", roleType).Scan(&roleTypeId); err == sql.ErrNoRows {
		return nil, InvalidRoleType
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password,role_type_id) VALUES (?, ?, ?)", email, string(hashedPassword), roleTypeId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tok, err := common.GenerateToken()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// store token in Token Database
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", tok, lastId, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return &AuthResponse{Token: tok, AccountId: lastId}, tx.Commit()
}

func (m *Auth) LogIn(email, password string) (*AuthResponse, error) {
	email = strings.ToLower(email)

	var accountId int64
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.DB.QueryRow("SELECT id, password FROM account WHERE email = ?", email).Scan(&accountId, &hashedPassword); err == sql.ErrNoRows {
		return nil, LoginDoesNotExist
	} else if err != nil {
		return nil, err
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := m.Hasher.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, InvalidPassword
	}

	token, err := common.GenerateToken()
	if err != nil {
		return nil, err
	}

	// delete any existing token and create a new one
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("DELETE FROM auth_token WHERE account_id = ?", accountId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// insert new token
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", token, accountId, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return &AuthResponse{Token: token, AccountId: accountId}, tx.Commit()
}

func (m *Auth) LogOut(token string) error {
	// delete the token from the database to invalidate
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE token = ?", token); err != nil {
		return err
	}
	return nil
}

func (m *Auth) ValidateToken(token string) (*TokenValidationResponse, error) {
	var accountId int64
	var expires time.Time
	if err := m.DB.QueryRow("SELECT account_id, expires FROM auth_token WHERE token = ?", token).Scan(&accountId, &expires); err == sql.ErrNoRows {
		return &TokenValidationResponse{IsValid: false, Reason: "token not found"}, nil
	} else if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// if the token exists, check the expiration to ensure that it is valid
	left := expires.Sub(now)
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
	return &TokenValidationResponse{IsValid: left > 0, AccountId: &accountId, Reason: reason}, nil
}

func (m *Auth) SetPassword(accountId int64, password string) error {
	if password == "" {
		return InvalidPassword
	}
	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return err
	}
	if res, err := m.DB.Exec("UPDATE account SET password = ? WHERE id = ?", string(hashedPassword), accountId); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	// Log out any existing tokens for the account
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE account_id = ?", accountId); err != nil {
		return err
	}
	return nil
}

func (m *Auth) UpdateLastOpenedDate(accountId int64) error {
	if res, err := m.DB.Exec(`update account set last_opened_date = now(6) where id = ?`, accountId); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	return nil
}
