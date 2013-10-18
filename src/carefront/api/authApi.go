package api

import (
	"code.google.com/p/go.crypto/bcrypt"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

const (
	EXPIRATION_THRESHOLD = 5 * 24 * 60 * time.Minute
)

type Account struct {
	Id       int64
	Email    string
	Password string
}

type AuthService struct {
	DB *sql.DB
}

func (m *AuthService) Signup(email, password string) (token string, accountId int64, err error) {
	// ensure to check that the email does not already exist in the database
	account := new(Account)
	err = m.DB.QueryRow("select * from Account where email = ?", email).Scan(&account.Id, &account.Email, &account.Password)
	if err == nil {
		return "", 0, ErrSignupFailedUserExists
	}

	// if its any error other than flagging the fact that no rows were returned,
	// inform calee
	if err != nil && err != sql.ErrNoRows {
		return "", 0, err
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", 0, err
	}

	// begin transaction to create an account
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	// create a new account since the user does not exist on the platform
	res, err := tx.Exec("insert into Account (email, password) values (?, ?)", email, string(hashedPassword))
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	tok, err := generateToken()
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	// store token in Token Database
	_, err = tx.Exec("insert into Token (token, account_id, created, expires) values (?, ?, ?, ?)", tok, lastId, time.Now(), time.Now().Add(EXPIRATION_THRESHOLD))
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	tx.Commit()
	return tok, lastId, nil
}

func (m *AuthService) Login(email, password string) (token string, accountId int64, err error) {
	var account Account

	// use the email address to lookup the Account from the table
	err = m.DB.QueryRow("select * from Account where email = ?", email).Scan(&account.Id, &account.Email, &account.Password)
	if err != nil {
		return "", 0, ErrLoginFailed
	}

	// compare the hashed password value to that stored in the database to authenticate the user
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
	if err != nil {
		return "", 0, err
	}

	// create new token
	token, err = generateToken()
	if err != nil {
		return "", 0, err
	}

	// delete any existing token and create a new one
	tx, err := m.DB.Begin()
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}
	// delete the token that exists (if one exists)
	_, err = tx.Exec("delete from Token where account_id = ?", account.Id)
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}

	// insert new token
	_, err = tx.Exec("insert into Token (token, account_id, created, expires) values (?, ?, ?, ?)", token, account.Id, time.Now(), time.Now().Add(EXPIRATION_THRESHOLD))
	if err != nil {
		tx.Rollback()
		return "", 0, err
	}
	tx.Commit()

	return token, account.Id, nil
}

func generateToken() (string, error) {

	// generate token for the new user and store token in database
	tokBytes := make([]byte, 16)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", err
	}

	tok := hex.EncodeToString(tokBytes)
	return tok, nil
}

func (m *AuthService) Logout(token string) error {

	// delete the token from the database to invalidate
	_, err := m.DB.Exec("delete from Token where token = ?", token)
	if err != nil {
		return err
	}
	return nil
}

func (m *AuthService) ValidateToken(token string) (valid bool, accountId int64, err error) {
	// lookup token in database
	var expires *time.Time
	err = m.DB.QueryRow("select account_id, expires from Token where token = ? ", token).Scan(&accountId, &expires)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, nil
		}
		return false, 0, err
	}

	// if the token exists, check the expiration to ensure that it is valid
	valid = time.Now().Before(*expires)
	return valid, accountId, nil
}
