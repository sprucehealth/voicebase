package api

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	InvalidPassword    = errors.New("api: invalid password")
	InvalidRoleType    = errors.New("api: invalid role type")
	LoginAlreadyExists = errors.New("api: login already exists")
	LoginDoesNotExist  = errors.New("api: login does not exist")
	TokenDoesNotExist  = errors.New("api: token does not exist")
	TokenExpired       = errors.New("api: token expired")
)

type authTokenConfig struct {
	expireDuration time.Duration
	renewDuration  time.Duration // When validating, if the time left on the token is less than this duration than the token is extended
}

type auth struct {
	regularAuth  *authTokenConfig
	extendedAuth *authTokenConfig
	db           *sql.DB
	hasher       PasswordHasher
	perms        map[int64]string
	permNames    map[string]int64
}

func normalizeEmail(email string) string {
	return strings.ToLower(email)
}

func NewAuthAPI(db *sql.DB, expireDuration, renewDuration, extendedAuthExpireDuration, extendedAuthRenewDuration time.Duration, hasher PasswordHasher) (AuthAPI, error) {
	ap := &auth{
		db: db,
		regularAuth: &authTokenConfig{
			renewDuration:  renewDuration,
			expireDuration: expireDuration,
		},
		extendedAuth: &authTokenConfig{
			renewDuration:  extendedAuthRenewDuration,
			expireDuration: extendedAuthExpireDuration,
		},
		hasher: hasher,
	}
	var err error
	ap.perms, err = ap.availableAccountPermissions()
	if err != nil {
		return nil, err
	}
	ap.permNames = make(map[string]int64, len(ap.perms))
	for id, name := range ap.perms {
		ap.permNames[name] = id
	}
	return ap, nil
}

func (m *auth) CreateAccount(email, password, roleType string) (int64, error) {
	if password == "" {
		return 0, InvalidPassword
	}
	email = normalizeEmail(email)

	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.db.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return 0, LoginAlreadyExists
	} else if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	hashedPassword, err := m.hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return 0, err
	}

	var roleTypeID int64
	if err := m.db.QueryRow("SELECT id from role_type where role_type_tag = ?", roleType).Scan(&roleTypeID); err == sql.ErrNoRows {
		return 0, InvalidRoleType
	}

	// begin transaction to create an account
	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password, role_type_id) VALUES (?, ?, ?)", email, string(hashedPassword), roleTypeID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return lastID, tx.Commit()
}

func (m *auth) Authenticate(email, password string, platform Platform, extendedAuth bool) (*common.Account, error) {
	email = normalizeEmail(email)

	var account common.Account
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.db.QueryRow(`
		SELECT account.id, role_type_tag, password, email
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE email = ?`, email,
	).Scan(&account.ID, &account.Role, &hashedPassword, &account.Email); err == sql.ErrNoRows {
		return nil, LoginDoesNotExist
	} else if err != nil {
		return nil, err
	}

	// identify an existing account config for this platform
	var accountExtendedAuth bool
	if err := m.db.QueryRow(`
		SELECT extended_auth 
		FROM account_config 
		WHERE account_id = ? AND platform = ?`,
		account.ID, string(platform)).Scan(&accountExtendedAuth); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if extendedAuth != accountExtendedAuth {
		if _, err := m.db.Exec(`
			REPLACE INTO account_config 
			(account_id, platform, extended_auth) VALUES (?,?,?)`,
			account.ID, string(platform), extendedAuth); err != nil {
			return nil, err
		}
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := m.hasher.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, InvalidPassword
	}

	return &account, nil
}

func (m *auth) CreateToken(accountID int64, platform Platform) (string, error) {
	token, err := common.GenerateToken()
	if err != nil {
		return "", err
	}

	// determine whether to treat as extendedAuth
	var extendedAuth bool
	if err := m.db.QueryRow(`
		SELECT extended_auth 
		FROM account_config WHERE account_id = ? and platform = ?`, accountID,
		string(platform)).Scan(&extendedAuth); err != sql.ErrNoRows && err != nil {
		return "", err
	}

	// delete any existing token and create a new one
	tx, err := m.db.Begin()
	if err != nil {
		return "", err
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("DELETE FROM auth_token WHERE account_id = ? AND platform = ?", accountID, string(platform))
	if err != nil {
		tx.Rollback()
		return "", err
	}

	// insert new token
	now := time.Now().UTC()
	expires := now.Add(m.regularAuth.expireDuration)
	if extendedAuth {
		expires = now.Add(m.extendedAuth.expireDuration)
	}

	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, platform, created, expires) VALUES (?, ?, ?, ?, ?)",
		token, accountID, string(platform), now, expires)
	if err != nil {
		tx.Rollback()
		return "", err
	}

	return token, tx.Commit()
}

func (m *auth) DeleteToken(token string) error {
	_, err := m.db.Exec("DELETE FROM auth_token WHERE token = ?", token)
	return err
}

func (m *auth) ValidateToken(token string, platform Platform) (*common.Account, error) {
	var account common.Account
	var expires time.Time
	var tokenPlatform string
	if err := m.db.QueryRow(`
		SELECT account_id, role_type_tag, expires, email, platform
		FROM auth_token
		INNER JOIN account ON account.id = account_id
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE token = ?`, token,
	).Scan(&account.ID, &account.Role, &expires, &account.Email, &tokenPlatform); err == sql.ErrNoRows {
		return nil, TokenDoesNotExist
	} else if err != nil {
		return nil, err
	}

	// determine whether the account is configured for extended auth on this platform
	var extendedAuth bool
	if err := m.db.QueryRow(`
		SELECT extended_auth 
		FROM account_config where account_id = ? AND platform = ?`,
		account.ID, string(platform)).
		Scan(&extendedAuth); err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if tokenPlatform != string(platform) {
		golog.Warningf("Platform does not match while validating token (expected %s got %+v)", tokenPlatform, platform)
		return nil, TokenDoesNotExist
	}

	// Check the expiration to ensure that it is valid
	now := time.Now().UTC()
	left := expires.Sub(now)
	if left <= 0 {
		golog.Infof("Current time %s is after expiration time %s", now.String(), expires.String())
		return nil, TokenExpired
	}
	// Extend token if necessary
	authConfig := m.regularAuth
	if extendedAuth {
		authConfig = m.extendedAuth
	}
	if authConfig.renewDuration > 0 && left < authConfig.renewDuration {
		if _, err := m.db.Exec("UPDATE auth_token SET expires = ? WHERE token = ?", now.Add(authConfig.expireDuration), token); err != nil {
			golog.Errorf("services/auth: failed to extend token expiration: %s", err.Error())
			// Don't return an error response because this doesn't prevent anything else from working
		}
	}

	return &account, nil
}

func (m *auth) GetToken(accountID int64) (string, error) {
	var token string
	err := m.db.QueryRow(`select token from auth_token where account_id = ?`, accountID).Scan(&token)
	if err == sql.ErrNoRows {
		return "", NoRowsError
	} else if err != nil {
		return "", err
	}

	return token, err
}

func (m *auth) SetPassword(accountID int64, password string) error {
	if password == "" {
		return InvalidPassword
	}
	hashedPassword, err := m.hasher.GenerateFromPassword([]byte(password))
	if err != nil {
		return err
	}
	if res, err := m.db.Exec("UPDATE account SET password = ? WHERE id = ?", string(hashedPassword), accountID); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	// Log out any existing tokens for the account
	if _, err := m.db.Exec("DELETE FROM auth_token WHERE account_id = ?", accountID); err != nil {
		return err
	}
	return nil
}

func (m *auth) UpdateLastOpenedDate(accountID int64) error {
	if res, err := m.db.Exec(`update account set last_opened_date = now(6) where id = ?`, accountID); err != nil {
		return err
	} else if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return NoRowsError
	}
	return nil
}

func (m *auth) GetPhoneNumbersForAccount(accountID int64) ([]*common.PhoneNumber, error) {
	rows, err := m.db.Query(`
		SELECT phone, phone_type, status
		FROM account_phone
		WHERE account_id = ? AND status = ?`, accountID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var numbers []*common.PhoneNumber
	for rows.Next() {
		num := &common.PhoneNumber{}
		if err := rows.Scan(&num.Phone, &num.Type, &num.Status); err != nil {
			return nil, err
		}
		numbers = append(numbers, num)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return numbers, nil
}

func (m *auth) GetAccountForEmail(email string) (*common.Account, error) {
	email = normalizeEmail(email)
	var account common.Account
	if err := m.db.QueryRow(`
		SELECT account.id, role_type_tag, email
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE email = ?`, email,
	).Scan(&account.ID, &account.Role, &account.Email); err == sql.ErrNoRows {
		return nil, LoginDoesNotExist
	} else if err != nil {
		return nil, err
	}
	return &account, nil
}

func (m *auth) GetAccount(id int64) (*common.Account, error) {
	account := &common.Account{
		ID: id,
	}
	if err := m.db.QueryRow(`
		SELECT role_type_tag, email
		FROM account
		INNER JOIN role_type ON role_type_id = role_type.id
		WHERE account.id = ?`, id,
	).Scan(&account.Role, &account.Email); err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return account, nil
}

func (m *auth) CreateTempToken(accountID int64, expireSec int, purpose, token string) (string, error) {
	if token == "" {
		var err error
		token, err = common.GenerateToken()
		if err != nil {
			return "", err
		}
	}
	expires := time.Now().Add(time.Duration(expireSec) * time.Second)
	_, err := m.db.Exec(`INSERT INTO temp_auth_token (token, purpose, account_id, expires) VALUES (?, ?, ?, ?)`,
		token, purpose, accountID, expires)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (m *auth) ValidateTempToken(purpose, token string) (*common.Account, error) {
	row := m.db.QueryRow(`
		SELECT expires, account_id, role_type_tag, email
		FROM temp_auth_token
		LEFT JOIN account ON account.id = account_id
		LEFT JOIN role_type ON role_type.id = account.role_type_id
		WHERE purpose = ? AND token = ?`, purpose, token)
	var expires time.Time
	var account common.Account
	if err := row.Scan(&expires, &account.ID, &account.Role, &account.Email); err == sql.ErrNoRows {
		return nil, TokenDoesNotExist
	} else if err != nil {
		return nil, err
	}
	if time.Now().After(expires) {
		return nil, TokenExpired
	}
	return &account, nil
}

func (m *auth) DeleteTempToken(purpose, token string) error {
	_, err := m.db.Exec(`DELETE FROM temp_auth_token WHERE token = ? AND purpose = ?`, token, purpose)
	return err
}

func (m *auth) DeleteTempTokensForAccount(accountID int64) error {
	_, err := m.db.Exec(`DELETE FROM temp_auth_token WHERE account_id = ?`, accountID)
	return err
}
