package api

import (
	"database/sql"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
)

// CreateToken creates a new token for the specific purpose and key. The key can be used
// by the caller to store what the token is protecting. For instance, it could be an account ID.
func (d *DataService) CreateToken(purpose, key, token string, expire time.Duration) (string, error) {
	if token == "" {
		var err error
		token, err = common.GenerateToken()
		if err != nil {
			return "", errors.Trace(err)
		}
	}
	expires := time.Now().Add(expire)
	_, err := d.db.Exec(`INSERT INTO "token" ("token", "purpose", "key", "expires") VALUES (?, ?, ?, ?)`,
		token, purpose, key, expires)
	if err != nil {
		return "", errors.Trace(err)
	}
	return token, nil
}

// ValidateToken returns the key for the token if it's valid. Otherwise it returns ErrTokenDoesNotExist
func (d *DataService) ValidateToken(purpose, token string) (string, error) {
	row := d.db.QueryRow(`
		SELECT "expires", "key"
		FROM "token"
		WHERE "purpose" = ? AND "token" = ?`, purpose, token)
	var key string
	var expires time.Time
	if err := row.Scan(
		&expires, &key,
	); err == sql.ErrNoRows {
		return "", ErrTokenDoesNotExist
	} else if err != nil {
		return "", errors.Trace(err)
	}
	if time.Now().After(expires) {
		return "", ErrTokenExpired
	}
	return key, nil
}

// DeleteToken deletes a specific token and returns number of rows deleted.
// It's useful for invalidation / revoking access.
func (d *DataService) DeleteToken(purpose, token string) (int, error) {
	res, err := d.db.Exec(`DELETE FROM "token" WHERE "token" = ? AND "purpose" = ?`, token, purpose)
	if err != nil {
		return 0, errors.Trace(err)
	}
	n, err := res.RowsAffected()
	return int(n), errors.Trace(err)
}
