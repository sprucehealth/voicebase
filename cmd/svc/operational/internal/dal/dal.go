package dal

import (
	"github.com/sprucehealth/backend/libs/errors"

	"database/sql"
)

type DAL interface {
	MarkAccountAsBlocked(email string) error
}

type dal struct {
	db *sql.DB
}

func NewDAL(db *sql.DB) DAL {
	return &dal{
		db: db,
	}
}

func (d *dal) MarkAccountAsBlocked(email string) error {
	_, err := d.db.Exec(`
		INSERT INTO blocked_accounts (email) VALUES (?)`, email)
	return errors.Trace(err)
}
