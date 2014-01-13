package auth

import (
	"database/sql"
	"log"
	"time"

	"carefront/common"
	"carefront/thrift/api"
	"code.google.com/p/go.crypto/bcrypt"
)

type AuthService struct {
	ExpireDuration time.Duration
	RenewDuration  time.Duration // When validation, if the time left on the token is less than this duration than the token is extended
	DB             *sql.DB
}

func (m *AuthService) Signup(email, password string) (*api.AuthResponse, error) {
	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.DB.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return nil, &api.LoginAlreadyExists{AccountId: id}
	} else if err != nil && err != sql.ErrNoRows {
		log.Printf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Printf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password) VALUES (?, ?)", email, string(hashedPassword))
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: INSERT account failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	tok, err := common.GenerateToken()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: GenerateToken failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// store token in Token Database
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", tok, lastId, time.Now(), time.Now().Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: INSERT auth_token failed: %s", err.Error())
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	tx.Commit()
	return &api.AuthResponse{Token: tok, AccountId: lastId}, nil
}

func (m *AuthService) Login(email, password string) (*api.AuthResponse, error) {
	var accountId int64
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.DB.QueryRow("SELECT id, password FROM account WHERE email = ?", email).Scan(&accountId, &hashedPassword); err == sql.ErrNoRows {
		return nil, &api.NoSuchLogin{}
	} else if err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
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
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", token, accountId, time.Now(), time.Now().Add(m.ExpireDuration))
	if err != nil {
		tx.Rollback()
		return nil, &api.InternalServerError{Message: err.Error()}
	}
	tx.Commit()

	return &api.AuthResponse{Token: token, AccountId: accountId}, nil
}

func (m *AuthService) Logout(token string) error {
	// delete the token from the database to invalidate
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE token = ?", token); err != nil {
		return &api.InternalServerError{Message: err.Error()}
	}
	return nil
}

func (m *AuthService) ValidateToken(token string) (*api.TokenValidationResponse, error) {
	var accountId int64
	var expires *time.Time
	if err := m.DB.QueryRow("SELECT account_id, expires FROM auth_token WHERE token =  ?", token).Scan(&accountId, &expires); err == sql.ErrNoRows {
		log.Printf("AUTHERROR: Token %s is not present in database ", token)
		return &api.TokenValidationResponse{IsValid: false}, nil
	} else if err != nil {
		return nil, &api.InternalServerError{Message: err.Error()}
	}

	// if the token exists, check the expiration to ensure that it is valid
	left := (*expires).Sub(time.Now())
	if left <= 0 {
		log.Printf("Current time %s is after expiration time %s", time.Now().String(), expires.String())
	} else if m.RenewDuration > 0 && left < m.RenewDuration {
		if _, err := m.DB.Exec("UPDATE auth_token SET expires = ? WHERE token = ?", time.Now().Add(m.ExpireDuration), token); err != nil {
			log.Printf("services/auth: failed to extend token expiration: %s", err.Error())
			// Don't return an error response because this doesn't prevent anything else from working
		}
	}
	return &api.TokenValidationResponse{IsValid: left > 0, AccountId: &accountId}, nil
}
