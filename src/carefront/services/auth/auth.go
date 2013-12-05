package auth

import (
	"database/sql"
	"log"
	"time"

	"carefront/common"
	"carefront/thriftapi"
	"code.google.com/p/go.crypto/bcrypt"
)

const (
	expirationThreshold = 5 * 24 * 60 * time.Minute
)

type AuthService struct {
	DB *sql.DB
}

func (m *AuthService) Signup(email, password string) (*thriftapi.AuthResponse, error) {
	// ensure to check that the email does not already exist in the database
	var id int64
	if err := m.DB.QueryRow("SELECT id FROM account WHERE email = ?", email).Scan(&id); err == nil {
		return nil, &thriftapi.LoginAlreadyExists{AccountId: id}
	} else if err != nil && err != sql.ErrNoRows {
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("INSERT INTO account (email, password) VALUES (?, ?)", email, string(hashedPassword))
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	tok, err := common.GenerateToken()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// store token in Token Database
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", tok, lastId, time.Now(), time.Now().Add(expirationThreshold))
	if err != nil {
		tx.Rollback()
		log.Printf("services/auth: %s", err.Error())
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	tx.Commit()
	return &thriftapi.AuthResponse{Token: tok, AccountId: lastId}, nil
}

func (m *AuthService) Login(email, password string) (*thriftapi.AuthResponse, error) {
	var accountId int64
	var hashedPassword string

	// use the email address to lookup the Account from the table
	if err := m.DB.QueryRow("SELECT id, password FROM account WHERE email = ?", email).Scan(&accountId, &hashedPassword); err == sql.ErrNoRows {
		return nil, &thriftapi.NoSuchLogin{}
	} else if err != nil {
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	token, err := common.GenerateToken()
	if err != nil {
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// delete any existing token and create a new one
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("DELETE FROM auth_token WHERE account_id = ?", accountId)
	if err != nil {
		tx.Rollback()
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// insert new token
	_, err = tx.Exec("INSERT INTO auth_token (token, account_id, created, expires) VALUES (?, ?, ?, ?)", token, accountId, time.Now(), time.Now().Add(expirationThreshold))
	if err != nil {
		tx.Rollback()
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}
	tx.Commit()

	return &thriftapi.AuthResponse{Token: token, AccountId: accountId}, nil
}

func (m *AuthService) Logout(token string) error {
	// delete the token from the database to invalidate
	if _, err := m.DB.Exec("DELETE FROM auth_token WHERE token = ?", token); err != nil {
		return &thriftapi.InternalServerError{Message: err.Error()}
	}
	return nil
}

func (m *AuthService) ValidateToken(token string) (*thriftapi.TokenValidationResponse, error) {
	var accountId int64
	var expires *time.Time
	if err := m.DB.QueryRow("SELECT account_id, expires FROM auth_token WHERE token = ? ", token).Scan(&accountId, &expires); err == sql.ErrNoRows {
		return &thriftapi.TokenValidationResponse{IsValid: false}, nil
	} else if err != nil {
		return nil, &thriftapi.InternalServerError{Message: err.Error()}
	}

	// if the token exists, check the expiration to ensure that it is valid
	valid := time.Now().Before(*expires)
	return &thriftapi.TokenValidationResponse{IsValid: valid, AccountId: &accountId}, nil
}
