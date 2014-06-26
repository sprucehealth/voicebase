package api

import (
	"database/sql"
	"errors"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"strings"
	"time"
)

var (
	InvalidPassword    = errors.New("api: invalid password")
	InvalidRoleType    = errors.New("api: invalid role type")
	LoginAlreadyExists = errors.New("api: login already exists")
	LoginDoesNotExist  = errors.New("api: login does not exist")
	TokenDoesNotExist  = errors.New("api: token does not exist")
	TokenExpired       = errors.New("api: token expired")
)

type Auth struct {
	ExpireDuration time.Duration
	RenewDuration  time.Duration // When validation, if the time left on the token is less than this duration than the token is extended
	DB             *sql.DB
	Hasher         PasswordHasher
}

func normalizeEmail(email string) string {
	return strings.ToLower(email)
}

func (m *Auth) SignUp(email, password, roleType string) (int64, string, error) {
	if password == "" {
		return 0, "", InvalidPassword
	}
	email = normalizeEmail(email)

	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.DB.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return 0, "", LoginAlreadyExists
	} else if err != nil && err != sql.ErrNoRows {
		return 0, "", err
	}

	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return 0, "", err
	}

	var roleTypeID int64
	if err := m.DB.QueryRow("SELECT id from role_type where role_type_tag = ?", roleType).Scan(&roleTypeID); err == sql.ErrNoRows {
		return 0, "", InvalidRoleType
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		return 0, "", err
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password, role_type_id) VALUES (?, ?, ?)", email, string(hashedPassword), roleTypeID)
	if err != nil {
		tx.Rollback()
		return 0, "", err
	}

	tok, err := common.GenerateToken()
	if err != nil {
		tx.Rollback()
		return 0, "", err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, "", err
	}

	// store token in Token Database
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", tok, lastID, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return 0, "", err
	}

	return lastID, tok, tx.Commit()
}

func (m *Auth) LogIn(email, password string) (*common.Account, string, error) {
	email = normalizeEmail(email)

	var account common.Account
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.DB.QueryRow(`
		SELECT account.id, role_type_tag, password
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE email = ?`, email,
	).Scan(&account.ID, &account.Role, &hashedPassword); err == sql.ErrNoRows {
		return nil, "", LoginDoesNotExist
	} else if err != nil {
		return nil, "", err
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := m.Hasher.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, "", InvalidPassword
	}

	token, err := common.GenerateToken()
	if err != nil {
		return nil, "", err
	}

	// delete any existing token and create a new one
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, "", err
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("DELETE FROM auth_token WHERE account_id = ?", account.ID)
	if err != nil {
		tx.Rollback()
		return nil, "", err
	}

	// insert new token
	now := time.Now().UTC()
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", token, account.ID, now, now.Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return nil, "", err
	}

	return &account, token, tx.Commit()
}

func (m *Auth) LogOut(token string) error {
	// delete the token from the database to invalidate
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE token = ?", token); err != nil {
		return err
	}
	return nil
}

func (m *Auth) ValidateToken(token string) (*common.Account, error) {
	var account common.Account
	var expires time.Time
	if err := m.DB.QueryRow(`
		SELECT account_id, role_type_tag, expires
		FROM auth_token
		INNER JOIN account ON account.id = account_id
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE token = ?`, token,
	).Scan(&account.ID, &account.Role, &expires); err == sql.ErrNoRows {
		return nil, TokenDoesNotExist
	} else if err != nil {
		return nil, err
	}

	// Check the expiration to ensure that it is valid
	now := time.Now().UTC()
	left := expires.Sub(now)
	if left <= 0 {
		golog.Infof("Current time %s is after expiration time %s", now.String(), expires.String())
		return nil, TokenExpired
	}
	// Extend token if necessary
	if m.RenewDuration > 0 && left < m.RenewDuration {
		if _, err := m.DB.Exec("UPDATE auth_token SET expires = ? WHERE token = ?", now.Add(m.ExpireDuration), token); err != nil {
			golog.Errorf("services/auth: failed to extend token expiration: %s", err.Error())
			// Don't return an error response because this doesn't prevent anything else from working
		}
	}

	return &account, nil
}

func (m *Auth) SetPassword(accountID int64, password string) error {
	if password == "" {
		return InvalidPassword
	}
	hashedPassword, err := m.Hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return err
	}
	if res, err := m.DB.Exec("UPDATE account SET password = ? WHERE id = ?", string(hashedPassword), accountID); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	// Log out any existing tokens for the account
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE account_id = ?", accountID); err != nil {
		return err
	}
	return nil
}

func (m *Auth) UpdateLastOpenedDate(accountID int64) error {
	if res, err := m.DB.Exec(`update account set last_opened_date = now(6) where id = ?`, accountID); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	return nil
}

func (m *Auth) GetAccountForEmail(email string) (*common.Account, error) {
	email = normalizeEmail(email)
	var account common.Account
	if err := m.DB.QueryRow(`
		SELECT account.id, role_type_tag
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE email = ?`, email,
	).Scan(&account.ID, &account.Role); err == sql.ErrNoRows {
		return nil, LoginDoesNotExist
	} else if err != nil {
		return nil, err
	}
	return &account, nil
}

func (m *Auth) GetAccount(id int64) (*common.Account, error) {
	account := &common.Account{
		ID: id,
	}
	if err := m.DB.QueryRow(`
		SELECT role_type_tag
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE account.id = ?`, id,
	).Scan(&account.Role); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return account, nil
}

func (m *Auth) CreateTempToken(accountID int64, expireSec int, purpose, token string) (string, error) {
	if token == "" {
		var err error
		token, err = common.GenerateToken()
		if err != nil {
			return "", err
		}
	}
	expires := time.Now().Add(time.Duration(expireSec) * time.Second)
	_, err := m.DB.Exec(`INSERT INTO temp_auth_token (token, purpose, account_id, expires) VALUES (?, ?, ?, ?)`,
		token, purpose, accountID, expires)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (m *Auth) ValidateTempToken(purpose, token string) (int64, string, error) {
	row := m.DB.QueryRow(`
		SELECT expires, account_id, role_type_tag
		FROM temp_auth_token
		LEFT JOIN account ON account.id = account_id
		LEFT JOIN role_type ON role_type.id = account.role_type_id
		WHERE purpose = ? AND token = ?`, purpose, token)
	var expires time.Time
	var accountID int64
	var roleType string
	if err := row.Scan(&expires, &accountID, &roleType); err == sql.ErrNoRows {
		return 0, "", TokenDoesNotExist
	} else if err != nil {
		return 0, "", err
	}
	if time.Now().After(expires) {
		return 0, "", TokenExpired
	}
	return accountID, roleType, nil
}

func (m *Auth) DeleteTempToken(purpose, token string) error {
	_, err := m.DB.Exec(`DELETE FROM temp_auth_token WHERE token = ? AND purpose = ?`, token, purpose)
	return err
}

func (m *Auth) DeleteTempTokensForAccount(accountID int64) error {
	_, err := m.DB.Exec(`DELETE FROM temp_auth_token WHERE account_id = ?`, accountID)
	return err
}
