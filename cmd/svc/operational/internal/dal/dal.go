package dal

import (
	"github.com/sprucehealth/backend/libs/errors"

	"database/sql"
)

type DAL interface {
	MarkAccountAsBlocked(accountID string) error
}

type dal struct {
	db *sql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: db,
	}
}

func (d *dal) MarkAccountAsBlocked(accountID string) error {
	_, err := d.db.Exec(`
		INSERT INTO blocked_accounts (account_id) VALUES (?)`, accountID)
	return errors.Trace(err)
}
